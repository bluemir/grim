package storage

import (
	"context"
	"crypto/sha256"
	"math/big"
	"time"

	"github.com/cockroachdb/errors"
	"gorm.io/gorm"

	"github.com/bluemir/grim/internal/util"
)

var (
	ErrStorageDisabled = errors.New("storage is disabled")
	ErrTooLarge        = errors.New("diagram source is too large")
	ErrRateLimited     = errors.New("too many diagrams created, try again later")
)

type Config struct {
	Enabled bool
	// MaxSize caps the source length of a stored diagram.
	MaxSize util.Size `yaml:"maxSize"`
	// Retention is how long a diagram survives without being viewed. 0 keeps it forever.
	Retention util.Duration
	RateLimit RateLimitConfig `yaml:"rateLimit"`
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
}

// Expiry describes when a diagram will be deleted, for display to viewers.
type Expiry struct {
	Pinned bool `json:"pinned"`
	// IdleDays is the retention window; 0 means diagrams never expire on this instance.
	IdleDays int `json:"idleDays"`
	// KeepUntil is set when an admin extension outlasts the idle window.
	KeepUntil *time.Time `json:"keepUntil,omitempty"`
}

const (
	idLength = 12
	// viewTouchInterval throttles LastViewedAt writes so every view isn't a DB write.
	viewTouchInterval = time.Hour
)

type Manager struct {
	db      *gorm.DB
	conf    Config
	limiter *rateLimiter
	now     func() time.Time
}

func New(ctx context.Context, conf *Config, db *gorm.DB) (*Manager, error) {
	m := &Manager{db: db, conf: *conf, now: time.Now}
	if conf.RateLimit.Create > 0 {
		m.limiter = newRateLimiter(conf.RateLimit.Create, conf.RateLimit.Window.Std())
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
	if err := m.db.WithContext(ctx).AutoMigrate(&Diagram{}); err != nil {
		return errors.WithStack(err)
	}
	err := m.db.WithContext(ctx).Model(&Diagram{}).
		Where("last_viewed_at IS NULL OR last_viewed_at = ?", time.Time{}).
		Update("last_viewed_at", m.now()).Error
	return errors.WithStack(err)
}

func (m *Manager) Enabled() bool { return m.conf.Enabled }

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
		if m.limiter != nil && !m.limiter.Allow(clientIP, now) {
			return nil, ErrRateLimited
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

func (m *Manager) Expiry(d *Diagram) Expiry {
	e := Expiry{Pinned: d.Pinned, IdleDays: m.conf.Retention.Days()}
	if d.KeepUntil != nil && d.KeepUntil.After(d.LastViewedAt.Add(m.conf.Retention.Std())) {
		e.KeepUntil = d.KeepUntil
	}
	return e
}

// ExpiresAt returns when the diagram will be deleted if nobody views it
// again. ok is false when it never expires.
func (m *Manager) ExpiresAt(d *Diagram) (at time.Time, ok bool) {
	if d.Pinned || m.conf.Retention == 0 {
		return time.Time{}, false
	}
	at = d.LastViewedAt.Add(m.conf.Retention.Std())
	if d.KeepUntil != nil && d.KeepUntil.After(at) {
		at = *d.KeepUntil
	}
	return at, true
}

// Cleanup deletes diagrams that are past both their idle window and any admin
// extension. It returns the number of deleted diagrams.
func (m *Manager) Cleanup(ctx context.Context) (int64, error) {
	now := m.now()
	if m.limiter != nil {
		m.limiter.Prune(now)
	}
	if !m.conf.Enabled || m.conf.Retention == 0 {
		return 0, nil
	}
	res := m.db.WithContext(ctx).
		Where("pinned = ?", false).
		Where("last_viewed_at < ?", now.Add(-m.conf.Retention.Std())).
		Where("keep_until IS NULL OR keep_until < ?", now).
		Delete(&Diagram{})
	return res.RowsAffected, errors.WithStack(res.Error)
}

// --- admin operations (used by the CLI) ---

func (m *Manager) List(ctx context.Context) ([]Diagram, error) {
	diagrams := []Diagram{}
	err := m.db.WithContext(ctx).
		Select("id", "created_at", "last_viewed_at", "keep_until", "pinned").
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
