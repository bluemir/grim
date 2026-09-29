package storage

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/bluemir/grim/internal/server/backend/mail"
	"github.com/bluemir/grim/internal/util"
)

var linkPattern = regexp.MustCompile(`https://grim\.example\.com/storage/contact#(\S+)`)

// tokens returns the tokens of all links with the given action in msg.
func tokens(t *testing.T, msg mail.Message, action string) []string {
	t.Helper()
	var out []string
	for _, m := range linkPattern.FindAllStringSubmatch(msg.Body, -1) {
		v, err := url.ParseQuery(m[1])
		require.NoError(t, err)
		if v.Get("action") == action {
			out = append(out, v.Get("token"))
		}
	}
	return out
}

func internalConfig(c *Config) {
	c.Retention = util.Duration(30 * day)
}

func verifiedDiagram(t *testing.T, m *Manager, f *fakeMailer, source string, emails ...string) *Diagram {
	t.Helper()
	ctx := context.Background()
	d, err := m.Create(ctx, source, "ip")
	require.NoError(t, err)
	for _, e := range emails {
		require.NoError(t, m.AddContact(ctx, d.Id, e))
		tok := tokens(t, f.sent[len(f.sent)-1], "confirm")
		require.Len(t, tok, 1)
		_, err := m.ConfirmContact(ctx, tok[0])
		require.NoError(t, err)
	}
	d, err = m.Find(ctx, d.Id)
	require.NoError(t, err)
	return d
}

func TestAddContactRequiresMail(t *testing.T) {
	m, _ := newTestManager(t, nil)
	d, _ := m.Create(context.Background(), "A", "ip")
	err := m.AddContact(context.Background(), d.Id, "a@example.com")
	assert.ErrorIs(t, err, mail.ErrMailDisabled)
}

func TestNewRequiresBaseURLWithMail(t *testing.T) {
	db, err := gorm.Open(sqliteMemory(), &gorm.Config{})
	require.NoError(t, err)
	conf := DefaultConfig()
	conf.Enabled = true
	_, err = New(context.Background(), &conf, db, &fakeMailer{enabled: true})
	assert.ErrorContains(t, err, "baseURL")
}

func TestVerifyKeepsDiagram(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()

	d, _ := m.Create(ctx, "A -> B", "ip")
	require.NoError(t, m.AddContact(ctx, d.Id, "  Alice@Example.com "))
	require.Len(t, f.sent, 1)
	assert.Equal(t, "alice@example.com", f.sent[0].To)
	assert.Contains(t, f.sent[0].Body, "https://grim.example.com/view/"+d.Id)

	// Not verified yet: an unverified contact doesn't change expiry or count.
	e, err := m.Expiry(ctx, d)
	require.NoError(t, err)
	assert.Equal(t, 0, e.Contacts)
	assert.Nil(t, e.VerifiedUntil)
	assert.True(t, e.CanVerify)

	res, err := m.ConfirmContact(ctx, tokens(t, f.sent[0], "confirm")[0])
	require.NoError(t, err)
	assert.Equal(t, d.Id, res.DiagramId)
	assert.True(t, res.VerifiedUntil.Equal(clock.t.Add(365*day)))

	d, _ = m.Find(ctx, d.Id)
	e, _ = m.Expiry(ctx, d)
	assert.Equal(t, 1, e.Contacts)

	// Far past the 30-day idle window but within the verified year.
	clock.advance(300 * day)
	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

func TestVerificationLinkExpires(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()

	d, _ := m.Create(ctx, "A", "ip")
	require.NoError(t, m.AddContact(ctx, d.Id, "a@example.com"))
	clock.advance(25 * time.Hour)

	_, err := m.ConfirmContact(ctx, tokens(t, f.sent[0], "confirm")[0])
	assert.ErrorIs(t, err, ErrLinkExpired)

	// Expired pending registrations are pruned.
	_, err = m.Cleanup(ctx)
	require.NoError(t, err)
	var n int64
	m.db.Model(&Contact{}).Count(&n)
	assert.EqualValues(t, 0, n)
}

func TestAddContactValidation(t *testing.T) {
	m, _, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d, _ := m.Create(ctx, "A", "ip")

	for _, bad := range []string{"", "not-an-email", "Bob <bob@example.com>", "a@b.com\r\nBcc: x@y.com"} {
		assert.ErrorIs(t, m.AddContact(ctx, d.Id, bad), ErrInvalidEmail, bad)
	}
	assert.ErrorIs(t, m.AddContact(ctx, "missing", "a@example.com"), gorm.ErrRecordNotFound)
	assert.Empty(t, f.sent)
}

func TestAddContactDailyLimit(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, func(c *Config) {
		internalConfig(c)
		c.Verification.DailyLimit = 2
	})
	ctx := context.Background()
	d1, _ := m.Create(ctx, "A", "ip")
	d2, _ := m.Create(ctx, "B", "ip")

	require.NoError(t, m.AddContact(ctx, d1.Id, "a@example.com"))
	require.NoError(t, m.AddContact(ctx, d2.Id, "a@example.com"))
	assert.ErrorIs(t, m.AddContact(ctx, d1.Id, "a@example.com"), ErrRateLimited)
	assert.NoError(t, m.AddContact(ctx, d1.Id, "b@example.com"), "limit is per address")

	clock.advance(24*time.Hour + time.Second)
	assert.NoError(t, m.AddContact(ctx, d1.Id, "a@example.com"))
	assert.Len(t, f.sent, 4)
}

func TestAddContactAlreadyVerifiedSendsNothing(t *testing.T) {
	m, _, f := newTestManagerWithMail(t, true, internalConfig)
	d := verifiedDiagram(t, m, f, "A", "a@example.com")
	sent := len(f.sent)
	require.NoError(t, m.AddContact(context.Background(), d.Id, "A@example.com"))
	assert.Len(t, f.sent, sent)
}

func TestReminderSchedule(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d := verifiedDiagram(t, m, f, "A", "a@example.com", "b@example.com")
	f.sent = nil

	sendAt := func(daysBeforeEnd int) int {
		clock.t = d.VerifiedUntil.Add(-time.Duration(daysBeforeEnd) * day)
		before := len(f.sent)
		require.NoError(t, m.SendReminders(ctx))
		require.NoError(t, m.SendReminders(ctx)) // running twice must not resend
		return len(f.sent) - before
	}

	assert.Equal(t, 0, sendAt(31))
	assert.Equal(t, 2, sendAt(30), "D-30: first mail, to every address")
	assert.Equal(t, 0, sendAt(26))
	assert.Equal(t, 2, sendAt(25))
	assert.Equal(t, 2, sendAt(20))
	assert.Equal(t, 2, sendAt(15))
	assert.Equal(t, 0, sendAt(10), "4 mails in total, the last at D-15")
	assert.Equal(t, 0, sendAt(1))
}

func TestReminderCatchUpSendsOnce(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d := verifiedDiagram(t, m, f, "A", "a@example.com")
	f.sent = nil

	// Server was down from before D-30 until D-18.
	clock.t = d.VerifiedUntil.Add(-18 * day)
	require.NoError(t, m.SendReminders(ctx))
	assert.Len(t, f.sent, 1)

	clock.t = d.VerifiedUntil.Add(-15 * day)
	require.NoError(t, m.SendReminders(ctx))
	assert.Len(t, f.sent, 2, "the D-15 mail still goes out")
}

func TestRemindersBatchedPerAddress(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d1 := verifiedDiagram(t, m, f, "A", "a@example.com")
	d2 := verifiedDiagram(t, m, f, "B", "a@example.com")
	f.sent = nil

	clock.t = d2.VerifiedUntil.Add(-30 * day)
	require.NoError(t, m.SendReminders(ctx))
	require.Len(t, f.sent, 1, "one mail listing both diagrams")
	assert.Contains(t, f.sent[0].Body, d1.Id)
	assert.Contains(t, f.sent[0].Body, d2.Id)
	assert.Len(t, tokens(t, f.sent[0], "confirm"), 2)
	assert.Len(t, tokens(t, f.sent[0], "remove"), 2)
}

func TestReminderRetriedWhenAllSendsFail(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d := verifiedDiagram(t, m, f, "A", "a@example.com")
	f.sent = nil

	clock.t = d.VerifiedUntil.Add(-30 * day)
	f.fail["a@example.com"] = true
	require.NoError(t, m.SendReminders(ctx))
	assert.Empty(t, f.sent)

	f.fail["a@example.com"] = false
	clock.advance(time.Hour)
	require.NoError(t, m.SendReminders(ctx))
	assert.Len(t, f.sent, 1)
}

func TestConfirmFromReminderExtends(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d := verifiedDiagram(t, m, f, "A", "a@example.com", "b@example.com")
	f.sent = nil

	clock.t = d.VerifiedUntil.Add(-30 * day)
	require.NoError(t, m.SendReminders(ctx))
	var bMail mail.Message
	for _, msg := range f.sent {
		if msg.To == "b@example.com" {
			bMail = msg
		}
	}

	// Only one of the two addresses answers.
	clock.advance(3 * day)
	res, err := m.ConfirmContact(ctx, tokens(t, bMail, "confirm")[0])
	require.NoError(t, err)
	assert.True(t, res.VerifiedUntil.Equal(clock.t.Add(365*day)))

	// The reminder cycle restarts for the new period.
	f.sent = nil
	clock.t = res.VerifiedUntil.Add(-30 * day)
	require.NoError(t, m.SendReminders(ctx))
	assert.Len(t, f.sent, 2)
}

func TestLapseWithoutResponse(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d := verifiedDiagram(t, m, f, "A", "a@example.com")

	clock.t = d.VerifiedUntil.Add(time.Minute)
	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "a lapse counts as a view, so it isn't deleted right away")

	d, err = m.Find(ctx, d.Id)
	require.NoError(t, err)
	assert.Nil(t, d.VerifiedUntil)
	e, _ := m.Expiry(ctx, d)
	assert.Equal(t, 0, e.Contacts, "contacts are dropped when the verification lapses")

	clock.advance(31 * day)
	n, err = m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n, "then the normal idle window applies")
}

func TestRemoveContact(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	ctx := context.Background()
	d := verifiedDiagram(t, m, f, "A", "a@example.com")
	f.sent = nil

	clock.t = d.VerifiedUntil.Add(-30 * day)
	require.NoError(t, m.SendReminders(ctx))
	res, err := m.RemoveContact(ctx, tokens(t, f.sent[0], "remove")[0])
	require.NoError(t, err)
	assert.Equal(t, d.Id, res.DiagramId)

	e, _ := m.Expiry(ctx, d)
	assert.Equal(t, 0, e.Contacts)

	_, err = m.RemoveContact(ctx, tokens(t, f.sent[0], "remove")[0])
	assert.ErrorIs(t, err, ErrLinkExpired, "the token is gone with the contact")

	f.sent = nil
	clock.advance(5 * day)
	require.NoError(t, m.SendReminders(ctx))
	assert.Empty(t, f.sent, "no more mail to the removed address")
}

func TestMailBodiesAreKorean(t *testing.T) {
	m, clock, f := newTestManagerWithMail(t, true, internalConfig)
	d := verifiedDiagram(t, m, f, "A", "a@example.com")
	assert.Contains(t, f.sent[0].Subject, "인증")

	clock.t = d.VerifiedUntil.Add(-30 * day)
	require.NoError(t, m.SendReminders(context.Background()))
	body := f.sent[len(f.sent)-1].Body
	assert.True(t, strings.Contains(body, "연장하기") && strings.Contains(body, "30일간 조회가 없으면 삭제"))
}
