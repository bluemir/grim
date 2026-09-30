package storage

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	netmail "net/mail"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/bluemir/grim/internal/server/backend/mail"
)

var (
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrInvalidLink means the token matches no registration: the address was
	// removed, re-registered with a newer link, or the diagram was deleted.
	ErrInvalidLink = errors.New("link is no longer valid")
)

// Contact is an email address registered on one diagram. There are no
// accounts: the token in mailed links is the only credential. Links stay
// valid as long as the registration exists; it is deleted with the diagram.
type Contact struct {
	Id         uint       `gorm:"primaryKey"`
	DiagramId  string     `gorm:"uniqueIndex:idx_contact_diagram_email;index"`
	Email      string     `gorm:"uniqueIndex:idx_contact_diagram_email"`
	Token      string     `gorm:"uniqueIndex"`
	VerifiedAt *time.Time // nil until the address owner confirms
	CreatedAt  time.Time
}

// MailLog records verification mails for the per-address daily limit.
type MailLog struct {
	Id     uint      `gorm:"primaryKey"`
	Email  string    `gorm:"index"`
	SentAt time.Time `gorm:"index"`
}

// ContactResult is what a confirm or remove link reports back to the page.
type ContactResult struct {
	DiagramId     string     `json:"diagramId"`
	VerifiedUntil *time.Time `json:"verifiedUntil,omitempty"`
}

// AddContact registers email on a diagram and mails a verification link.
// Anyone who can view the diagram may do this; the diagram itself never changes.
func (m *Manager) AddContact(ctx context.Context, diagramID, email string) error {
	if !m.VerificationEnabled() {
		return mail.ErrMailDisabled
	}
	addr, err := netmail.ParseAddress(strings.TrimSpace(email))
	if err != nil || addr.Name != "" {
		return ErrInvalidEmail
	}
	email = strings.ToLower(addr.Address)

	d, err := m.Find(ctx, diagramID)
	if err != nil {
		return err
	}

	now := m.now()
	contact := &Contact{}
	err = m.db.WithContext(ctx).Take(contact, "diagram_id = ? AND email = ?", d.Id, email).Error
	switch {
	case err == nil && contact.VerifiedAt != nil:
		return nil // already verified; nothing to send
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		contact = &Contact{DiagramId: d.Id, Email: email, CreatedAt: now}
	default:
		return errors.WithStack(err)
	}

	if err := m.checkDailyLimit(ctx, email, now); err != nil {
		return err
	}

	token, err := newToken()
	if err != nil {
		return err
	}
	contact.Token = token // replaces any earlier link for this address
	if err := m.db.WithContext(ctx).Save(contact).Error; err != nil {
		return errors.WithStack(err)
	}

	if err := m.mailer.Send(ctx, m.verificationMail(d, contact, now)); err != nil {
		return errors.Wrap(err, "send verification mail")
	}
	return errors.WithStack(m.db.WithContext(ctx).Create(&MailLog{Email: email, SentAt: now}).Error)
}

func (m *Manager) checkDailyLimit(ctx context.Context, email string, now time.Time) error {
	limit := m.conf.Verification.DailyLimit
	if limit <= 0 {
		return nil
	}
	var n int64
	err := m.db.WithContext(ctx).Model(&MailLog{}).
		Where("email = ? AND sent_at > ?", email, now.Add(-24*time.Hour)).Count(&n).Error
	if err != nil {
		return errors.WithStack(err)
	}
	if n >= int64(limit) {
		return ErrRateLimited
	}
	return nil
}

// ConfirmContact handles the link in both the verification mail and the
// reminder mails: it verifies the address if needed and keeps the diagram for
// another Verification.Retention from now.
func (m *Manager) ConfirmContact(ctx context.Context, token string) (*ContactResult, error) {
	if !m.conf.Enabled {
		return nil, ErrStorageDisabled
	}
	now := m.now()
	contact, err := m.contactByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	until := now.Add(m.conf.Verification.Retention.Std())
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if contact.VerifiedAt == nil {
			if err := tx.Model(contact).Update("verified_at", now).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Diagram{}).Where("id = ?", contact.DiagramId).
			Updates(map[string]any{"verified_until": until, "reminders_sent": 0}).Error
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &ContactResult{DiagramId: contact.DiagramId, VerifiedUntil: &until}, nil
}

// RemoveContact deletes the address behind token from its diagram. The
// diagram keeps any verification period it already has.
func (m *Manager) RemoveContact(ctx context.Context, token string) (*ContactResult, error) {
	if !m.conf.Enabled {
		return nil, ErrStorageDisabled
	}
	contact, err := m.contactByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := m.db.WithContext(ctx).Delete(contact).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return &ContactResult{DiagramId: contact.DiagramId}, nil
}

func (m *Manager) contactByToken(ctx context.Context, token string) (*Contact, error) {
	contact := &Contact{}
	if token == "" {
		return nil, ErrInvalidLink
	}
	err := m.db.WithContext(ctx).Take(contact, "token = ?", token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidLink
	}
	return contact, errors.WithStack(err)
}

// SendReminders mails every verified address whose diagrams are nearing the
// end of their verification period. Mails are batched per address, so one
// person with many diagrams gets one mail per round.
func (m *Manager) SendReminders(ctx context.Context) error {
	if !m.VerificationEnabled() {
		return nil
	}
	now := m.now()
	v := m.conf.Verification
	if v.Reminders <= 0 {
		return nil
	}

	var candidates []Diagram
	err := m.db.WithContext(ctx).
		Select("id", "verified_until", "reminders_sent").
		Where("verified_until IS NOT NULL AND verified_until > ? AND reminders_sent < ?", now, v.Reminders).
		Where("verified_until <= ?", now.Add(v.RemindBefore.Std())).
		Find(&candidates).Error
	if err != nil {
		return errors.WithStack(err)
	}

	// due maps diagram id -> the RemindersSent value after this round.
	due := map[string]int{}
	byID := map[string]*Diagram{}
	for i := range candidates {
		d := &candidates[i]
		if sent := remindersDue(d, now, v); sent > d.RemindersSent {
			due[d.Id] = sent
			byID[d.Id] = d
		}
	}
	if len(due) == 0 {
		return nil
	}

	ids := make([]string, 0, len(due))
	for id := range due {
		ids = append(ids, id)
	}
	var contacts []Contact
	err = m.db.WithContext(ctx).
		Where("diagram_id IN ? AND verified_at IS NOT NULL", ids).
		Order("email, diagram_id").Find(&contacts).Error
	if err != nil {
		return errors.WithStack(err)
	}
	byEmail := map[string][]Contact{}
	for _, c := range contacts {
		byEmail[c.Email] = append(byEmail[c.Email], c)
	}

	// A diagram's round counts as done once any of its addresses got the mail
	// (or it has none to mail), so a single bad address doesn't cause resends.
	delivered := map[string]bool{}
	for email, cs := range byEmail {
		if err := m.mailer.Send(ctx, m.reminderMail(email, cs, byID)); err != nil {
			logrus.Warnf("send reminder to %s: %v", email, err)
			continue
		}
		for _, c := range cs {
			delivered[c.DiagramId] = true
		}
	}
	hasContact := map[string]bool{}
	for _, c := range contacts {
		hasContact[c.DiagramId] = true
	}

	for id, sent := range due {
		if hasContact[id] && !delivered[id] {
			continue
		}
		err := m.db.WithContext(ctx).Model(&Diagram{}).Where("id = ?", id).
			Update("reminders_sent", sent).Error
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// remindersDue returns how many reminders should have gone out by now for d:
// the first at VerifiedUntil-RemindBefore, then one every RemindInterval.
// Reminders missed while the server was down collapse into one mail.
func remindersDue(d *Diagram, now time.Time, v VerificationConfig) int {
	first := d.VerifiedUntil.Add(-v.RemindBefore.Std())
	if now.Before(first) {
		return 0
	}
	n := 1
	if v.RemindInterval > 0 {
		n += int(now.Sub(first) / v.RemindInterval.Std())
	}
	return min(n, v.Reminders)
}

// lapseVerifications turns diagrams whose verification period ended without a
// confirmation back into ordinary diagrams. The lapse counts as a view, so the
// diagram gets the full idle window before it can be deleted.
func (m *Manager) lapseVerifications(ctx context.Context, now time.Time) error {
	var ids []string
	err := m.db.WithContext(ctx).Model(&Diagram{}).
		Where("verified_until IS NOT NULL AND verified_until <= ?", now).
		Pluck("id", &ids).Error
	if err != nil || len(ids) == 0 {
		return errors.WithStack(err)
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("diagram_id IN ?", ids).Delete(&Contact{}).Error; err != nil {
			return err
		}
		return tx.Model(&Diagram{}).Where("id IN ?", ids).Updates(map[string]any{
			"verified_until": nil,
			"reminders_sent": 0,
			"last_viewed_at": now,
		}).Error
	})
	return errors.WithStack(err)
}

// pruneMailLogs drops mail logs older than the daily-limit window.
func (m *Manager) pruneMailLogs(ctx context.Context, now time.Time) error {
	err := m.db.WithContext(ctx).Where("sent_at < ?", now.Add(-24*time.Hour)).Delete(&MailLog{}).Error
	return errors.WithStack(err)
}

// --- mail content ---

const dateLayout = "2006-01-02"

func (m *Manager) link(path string) string {
	return strings.TrimRight(m.conf.BaseURL, "/") + path
}

func (m *Manager) contactLink(action, token string) string {
	return m.link("/storage/contact#" + url.Values{"action": {action}, "token": {token}}.Encode())
}

func (m *Manager) verificationMail(d *Diagram, c *Contact, now time.Time) mail.Message {
	v := m.conf.Verification
	var b strings.Builder
	fmt.Fprintf(&b, "grim 다이어그램의 보관 기간 연장을 위해 이 메일 주소가 등록되었습니다.\n\n")
	fmt.Fprintf(&b, "다이어그램: %s\n\n", m.link("/view/"+d.Id))
	fmt.Fprintf(&b, "아래 링크에서 인증하면 %s까지 보관됩니다.\n", now.Add(v.Retention.Std()).Format(dateLayout))
	fmt.Fprintf(&b, "보관 기간이 끝나기 %d일 전부터 연장 확인 메일을 보내 드립니다.\n\n", v.RemindBefore.Days())
	fmt.Fprintf(&b, "인증하기: %s\n", m.contactLink("confirm", c.Token))
	fmt.Fprintf(&b, "이 링크는 다이어그램이 삭제되기 전까지 유효합니다.\n\n")
	fmt.Fprintf(&b, "직접 등록하지 않았다면 이 메일을 무시하세요. 인증하지 않으면 주소는 저장되지 않습니다.\n")
	return mail.Message{To: c.Email, Subject: "[grim] 다이어그램 보관 메일 인증", Body: b.String()}
}

func (m *Manager) reminderMail(email string, contacts []Contact, diagrams map[string]*Diagram) mail.Message {
	sort.Slice(contacts, func(i, j int) bool {
		return diagrams[contacts[i].DiagramId].VerifiedUntil.Before(*diagrams[contacts[j].DiagramId].VerifiedUntil)
	})
	var b strings.Builder
	fmt.Fprintf(&b, "등록하신 grim 다이어그램의 보관 기간이 곧 끝납니다.\n")
	fmt.Fprintf(&b, "계속 보관하려면 각 다이어그램의 \"연장하기\" 링크를 눌러 주세요. %d일 더 보관됩니다.\n",
		m.conf.Verification.Retention.Days())
	if m.conf.Retention > 0 {
		fmt.Fprintf(&b, "연장하지 않으면 보관 기간이 끝난 뒤 일반 다이어그램으로 바뀌며, %d일간 조회가 없으면 삭제됩니다.\n",
			m.conf.Retention.Days())
	}
	b.WriteString("\n")
	for _, c := range contacts {
		d := diagrams[c.DiagramId]
		fmt.Fprintf(&b, "- %s (보관 기한: %s)\n", m.link("/view/"+d.Id), d.VerifiedUntil.Format(dateLayout))
		fmt.Fprintf(&b, "  연장하기: %s\n", m.contactLink("confirm", c.Token))
		fmt.Fprintf(&b, "  이 다이어그램에서 내 메일 삭제: %s\n\n", m.contactLink("remove", c.Token))
	}
	return mail.Message{To: email, Subject: "[grim] 다이어그램 보관 기간 만료 예정 안내", Body: b.String()}
}

func newToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.WithStack(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
