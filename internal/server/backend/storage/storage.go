package storage

import (
	"context"
	"crypto/sha256"
	"math/big"
	"time"

	"github.com/cockroachdb/errors"
	"gorm.io/gorm"

	"github.com/bluemir/grim/internal/server/backend/mail"
	"github.com/bluemir/grim/internal/util"
)

var (
	ErrStorageDisabled = errors.New("storage is disabled")
	ErrTooLarge        = errors.New("diagram source is too large")
	ErrRateLimited     = errors.New("too many requests, try again later")
)

type Config struct {
	Enabled bool
	// MaxSize caps the source length of a stored diagram.
	MaxSize util.Size `yaml:"maxSize"`
	// Retention is how long a diagram survives without being viewed. 0 keeps it forever.
	Retention util.Duration
	RateLimit RateLimitConfig `yaml:"rateLimit"`
	// Verification applies when mail is configured: diagrams whose viewers
	// register and confirm an email address are kept longer.
	Verification VerificationConfig `yaml:"verification"`

	// BaseURL is the public server URL used in mailed links. It comes from http.baseURL.
	BaseURL string `yaml:"-" json:"-"`
}

type VerificationConfig struct {
	// Retention is how long a verified diagram is kept after the last confirmation.
	Retention util.Duration
	// RemindBefore is how long before the end of Retention the first confirmation mail goes out.
	RemindBefore util.Duration `yaml:"remindBefore"`
	// RemindInterval is the gap between confirmation mails.
	RemindInterval util.Duration `yaml:"remindInterval"`
	// Reminders is the total number of confirmation mails, including the first.
	Reminders int
	// LinkTTL is how long the link in a verification mail stays valid.
	LinkTTL util.Duration `yaml:"linkTTL"`
	// DailyLimit caps verification mails sent to one address per 24 hours. 0 disables the limit.
	DailyLimit int `yaml:"dailyLimit"`
}

type RateLimitConfig struct {
	// Create is the number of diagrams one client IP may store per Window. 0 disables the limit.
	Create int
	Window util.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxSize:   1 << 20,
		Retention: util.Duration(90 * 24 * time.Hour),
		RateLimit: RateLimitConfig{
			Create: 30,
			Window: util.Duration(time.Hour),
		},
		Verification: VerificationConfig{
			Retention:      util.Duration(365 * 24 * time.Hour),
			RemindBefore:   util.Duration(30 * 24 * time.Hour),
			RemindInterval: util.Duration(5 * 24 * time.Hour),
			Reminders:      4,
			LinkTTL:        util.Duration(24 * time.Hour),
			DailyLimit:     30,
		},
	}
}

// Diagram is an immutable snapshot. Its Id is derived from Source, so storing
// the same source twice yields the same diagram.
type Diagram struct {
	Id           string     `json:"id" gorm:"primaryKey"`
	Source       string     `json:"source" gorm:"type:text"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastViewedAt time.Time  `json:"lastViewedAt" gorm:"index"`
	KeepUntil    *time.Time `json:"keepUntil,omitempty"` // set by an admin; the diagram survives at least until then
	Pinned       bool       `json:"pinned"`              // set by an admin; never expires

	// VerifiedUntil is set while at least one registered email has confirmed;
	// the diagram is kept until then regardless of views.
	VerifiedUntil *time.Time `json:"verifiedUntil,omitempty"`
	// RemindersSent counts confirmation mails sent for the current VerifiedUntil.
	RemindersSent int `json:"-"`
}

// Expiry describes when a diagram will be deleted, for display to viewers.
type Expiry struct {
	Pinned bool `json:"pinned"`
	// IdleDays is the retention window; 0 means diagrams never expire on this instance.
	IdleDays int `json:"idleDays"`
	// KeepUntil is set when an admin extension outlasts the idle window.
	KeepUntil *time.Time `json:"keepUntil,omitempty"`
	// VerifiedUntil is set while the diagram is kept by email verification.
	VerifiedUntil *time.Time `json:"verifiedUntil,omitempty"`
	// Contacts is the number of confirmed email addresses. Addresses are never exposed.
	Contacts int `json:"contacts"`
	// CanVerify tells the viewer whether it may offer email registration.
	CanVerify bool `json:"canVerify"`
}

const (
	idLength = 12
	// viewTouchInterval throttles LastViewedAt writes so every view isn't a DB write.
	viewTouchInterval = time.Hour
)

type Manager struct {
	db      *gorm.DB
	conf    Config
	mailer  mail.Sender
	limiter *rateLimiter
	now     func() time.Time
}

func New(ctx context.Context, conf *Config, db *gorm.DB, mailer mail.Sender) (*Manager, error) {
	if mailer.Enabled() && conf.Enabled && conf.BaseURL == "" {
		return nil, errors.New("http.baseURL is required when mail is configured, for links in mails")
	}
	m := &Manager{db: db, conf: *conf, mailer: mailer, now: time.Now}
	if conf.RateLimit.Create > 0 {
		// m.now is read on each call so tests can swap the clock after New.
		limiter, err := newRateLimiter(conf.RateLimit.Create, conf.RateLimit.Window.Std(), rateLimitMaxKeys, func() time.Time { return m.now() })
		if err != nil {
			return nil, err
		}
		m.limiter = limiter
	}
	if conf.Enabled {
		if err := m.Migrate(ctx); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Migrate creates or upgrades the diagrams table. Diagrams saved before view
// tracking existed get the migration time as their last view, so they don't
// expire the moment this version is deployed.
func (m *Manager) Migrate(ctx context.Context) error {
	if err := m.db.WithContext(ctx).AutoMigrate(&Diagram{}, &Contact{}, &MailLog{}); err != nil {
		return errors.WithStack(err)
	}
	err := m.db.WithContext(ctx).Model(&Diagram{}).
		Where("last_viewed_at IS NULL OR last_viewed_at = ?", time.Time{}).
		Update("last_viewed_at", m.now()).Error
	return errors.WithStack(err)
}

func (m *Manager) Enabled() bool { return m.conf.Enabled }

// VerificationEnabled reports whether viewers can register emails to keep diagrams longer.
func (m *Manager) VerificationEnabled() bool { return m.conf.Enabled && m.mailer.Enabled() }

// MaxRequestBytes bounds the create request body: the source limit plus room
// for JSON escaping. 0 means unlimited.
func (m *Manager) MaxRequestBytes() int64 {
	if m.conf.MaxSize <= 0 {
		return 0
	}
	return int64(m.conf.MaxSize)*2 + 4096
}

// Create stores source and returns its snapshot, or the existing one if the
// same source was stored before. clientIP is used for rate limiting.
func (m *Manager) Create(ctx context.Context, source string, clientIP string) (*Diagram, error) {
	if !m.conf.Enabled {
		return nil, ErrStorageDisabled
	}
	if m.conf.MaxSize > 0 && int64(len(source)) > int64(m.conf.MaxSize) {
		return nil, errors.Wrapf(ErrTooLarge, "limit is %s", m.conf.MaxSize)
	}
	full := hashID(source)
	now := m.now()
	// On the (astronomically unlikely) chance that a prefix collides with a
	// different source, fall back to a longer prefix of the same hash.
	for n := idLength; n <= len(full); n++ {
		id := full[:n]
		existing := &Diagram{}
		err := m.db.WithContext(ctx).Take(existing, "id = ?", id).Error
		switch {
		case err == nil && existing.Source == source:
			// Re-sharing counts as a view.
			if err := m.touch(ctx, existing, now); err != nil {
				return nil, err
			}
			return existing, nil
		case err == nil:
			continue
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return nil, errors.WithStack(err)
		}

		// Only new rows count against the limit; re-sharing stores nothing.
		if m.limiter != nil {
			ok, err := m.limiter.Allow(ctx, clientIP)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, ErrRateLimited
			}
		}
		diagram := &Diagram{Id: id, Source: source, CreatedAt: now, LastViewedAt: now}
		if err := m.db.WithContext(ctx).Create(diagram).Error; err != nil {
			return nil, errors.WithStack(err)
		}
		return diagram, nil
	}
	return nil, errors.New("unable to allocate diagram id")
}

// Get returns a diagram and records the view, which resets its idle timer.
func (m *Manager) Get(ctx context.Context, id string) (*Diagram, error) {
	diagram, err := m.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := m.touch(ctx, diagram, m.now()); err != nil {
		return nil, err
	}
	return diagram, nil
}

// Find returns a diagram without recording a view.
func (m *Manager) Find(ctx context.Context, id string) (*Diagram, error) {
	if !m.conf.Enabled {
		return nil, ErrStorageDisabled
	}
	diagram := &Diagram{}
	if err := m.db.WithContext(ctx).Take(diagram, "id = ?", id).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return diagram, nil
}

func (m *Manager) touch(ctx context.Context, d *Diagram, now time.Time) error {
	if now.Sub(d.LastViewedAt) < viewTouchInterval {
		return nil
	}
	d.LastViewedAt = now
	err := m.db.WithContext(ctx).Model(d).Update("last_viewed_at", now).Error
	return errors.WithStack(err)
}

func (m *Manager) Expiry(ctx context.Context, d *Diagram) (Expiry, error) {
	e := Expiry{
		Pinned:        d.Pinned,
		IdleDays:      m.conf.Retention.Days(),
		VerifiedUntil: d.VerifiedUntil,
		CanVerify:     m.VerificationEnabled(),
	}
	if d.KeepUntil != nil && d.KeepUntil.After(d.LastViewedAt.Add(m.conf.Retention.Std())) {
		e.KeepUntil = d.KeepUntil
	}
	var n int64
	err := m.db.WithContext(ctx).Model(&Contact{}).
		Where("diagram_id = ? AND verified_at IS NOT NULL", d.Id).Count(&n).Error
	e.Contacts = int(n)
	return e, errors.WithStack(err)
}

// ExpiresAt returns when the diagram will be deleted if nobody views it
// again. ok is false when it never expires.
func (m *Manager) ExpiresAt(d *Diagram) (at time.Time, ok bool) {
	if d.Pinned || m.conf.Retention == 0 {
		return time.Time{}, false
	}
	at = d.LastViewedAt.Add(m.conf.Retention.Std())
	for _, t := range []*time.Time{d.KeepUntil, d.VerifiedUntil} {
		if t != nil && t.After(at) {
			at = *t
		}
	}
	return at, true
}

// Cleanup deletes diagrams that are past their idle window, any admin
// extension and any verification period. Verifications that lapsed without a
// confirmation are turned back into ordinary diagrams first. It returns the
// number of deleted diagrams.
func (m *Manager) Cleanup(ctx context.Context) (int64, error) {
	now := m.now()
	if !m.conf.Enabled {
		return 0, nil
	}
	if err := m.lapseVerifications(ctx, now); err != nil {
		return 0, err
	}
	if err := m.pruneContactsAndLogs(ctx, now); err != nil {
		return 0, err
	}
	if m.conf.Retention == 0 {
		return 0, nil
	}
	var ids []string
	err := m.db.WithContext(ctx).Model(&Diagram{}).
		Where("pinned = ?", false).
		Where("last_viewed_at < ?", now.Add(-m.conf.Retention.Std())).
		Where("keep_until IS NULL OR keep_until < ?", now).
		Where("verified_until IS NULL OR verified_until < ?", now).
		Pluck("id", &ids).Error
	if err != nil || len(ids) == 0 {
		return 0, errors.WithStack(err)
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("diagram_id IN ?", ids).Delete(&Contact{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&Diagram{}).Error
	})
	return int64(len(ids)), errors.WithStack(err)
}

// --- admin operations (used by the CLI) ---

func (m *Manager) List(ctx context.Context) ([]Diagram, error) {
	diagrams := []Diagram{}
	err := m.db.WithContext(ctx).
		Select("id", "created_at", "last_viewed_at", "keep_until", "pinned", "verified_until").
		Order("created_at").Find(&diagrams).Error
	return diagrams, errors.WithStack(err)
}

func (m *Manager) SetPinned(ctx context.Context, id string, pinned bool) error {
	return m.updateOne(ctx, id, "pinned", pinned)
}

// Extend keeps a diagram until at least now+d, whether or not it is viewed.
func (m *Manager) Extend(ctx context.Context, id string, d time.Duration) (time.Time, error) {
	until := m.now().Add(d)
	return until, m.updateOne(ctx, id, "keep_until", until)
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	if err := m.db.WithContext(ctx).Delete(&Contact{}, "diagram_id = ?", id).Error; err != nil {
		return errors.WithStack(err)
	}
	res := m.db.WithContext(ctx).Delete(&Diagram{}, "id = ?", id)
	if res.Error != nil {
		return errors.WithStack(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.WithStack(gorm.ErrRecordNotFound)
	}
	return nil
}

func (m *Manager) updateOne(ctx context.Context, id string, column string, value any) error {
	res := m.db.WithContext(ctx).Model(&Diagram{}).Where("id = ?", id).Update(column, value)
	if res.Error != nil {
		return errors.WithStack(res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.WithStack(gorm.ErrRecordNotFound)
	}
	return nil
}

// hashID returns the sha256 of source in base62.
func hashID(source string) string {
	sum := sha256.Sum256([]byte(source))
	return new(big.Int).SetBytes(sum[:]).Text(62)
}
