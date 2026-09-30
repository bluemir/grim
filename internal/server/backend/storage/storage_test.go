package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/bluemir/grim/internal/server/backend/mail"
	"github.com/bluemir/grim/internal/util"
)

const day = 24 * time.Hour

type testClock struct{ t time.Time }

func (c *testClock) now() time.Time          { return c.t }
func (c *testClock) advance(d time.Duration) { c.t = c.t.Add(d) }

// fakeMailer records messages instead of sending them.
type fakeMailer struct {
	enabled bool
	sent    []mail.Message
	fail    map[string]bool // recipient -> fail sending
}

func (f *fakeMailer) Enabled() bool { return f.enabled }
func (f *fakeMailer) Send(_ context.Context, msg mail.Message) error {
	if err := msg.Validate(); err != nil {
		return err // the real sender would reject it too
	}
	if f.fail[msg.To] {
		return errors.New("smtp: mailbox unavailable")
	}
	f.sent = append(f.sent, msg)
	return nil
}

func newTestManager(t *testing.T, mutate func(*Config)) (*Manager, *testClock) {
	m, clock, _ := newTestManagerWithMail(t, false, mutate)
	return m, clock
}

func newTestManagerWithMail(t *testing.T, mailEnabled bool, mutate func(*Config)) (*Manager, *testClock, *fakeMailer) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // keep a single :memory: database

	conf := DefaultConfig()
	conf.Enabled = true
	conf.RateLimit.Create = 0
	conf.BaseURL = "https://grim.example.com"
	if mutate != nil {
		mutate(&conf)
	}
	mailer := &fakeMailer{enabled: mailEnabled, fail: map[string]bool{}}
	m, err := New(context.Background(), &conf, db, mailer)
	require.NoError(t, err)

	clock := &testClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	m.now = clock.now
	return m, clock, mailer
}

func TestCreateIsContentAddressed(t *testing.T) {
	m, _ := newTestManager(t, nil)
	ctx := context.Background()

	a, err := m.Create(ctx, "A -> B", "1.1.1.1")
	require.NoError(t, err)
	again, err := m.Create(ctx, "A -> B", "1.1.1.1")
	require.NoError(t, err)
	other, err := m.Create(ctx, "A -> C", "1.1.1.1")
	require.NoError(t, err)

	assert.Len(t, a.Id, idLength)
	assert.Equal(t, a.Id, again.Id, "same source must map to the same id")
	assert.NotEqual(t, a.Id, other.Id)

	got, err := m.Get(ctx, a.Id)
	require.NoError(t, err)
	assert.Equal(t, "A -> B", got.Source)
}

func TestCreateHandlesPrefixCollision(t *testing.T) {
	m, _ := newTestManager(t, nil)
	ctx := context.Background()

	// Occupy the 12-char prefix of "A -> B" with a different source.
	id := hashID("A -> B")[:idLength]
	require.NoError(t, m.db.Create(&Diagram{Id: id, Source: "something else"}).Error)

	d, err := m.Create(ctx, "A -> B", "1.1.1.1")
	require.NoError(t, err)
	assert.Equal(t, hashID("A -> B")[:idLength+1], d.Id)
}

func TestCreateRejectsTooLarge(t *testing.T) {
	m, _ := newTestManager(t, func(c *Config) { c.MaxSize = 10 })

	_, err := m.Create(context.Background(), strings.Repeat("x", 11), "1.1.1.1")
	assert.ErrorIs(t, err, ErrTooLarge)
}

func TestCreateRateLimit(t *testing.T) {
	m, clock := newTestManager(t, func(c *Config) {
		c.RateLimit = RateLimitConfig{Create: 2, Window: util.Duration(time.Hour)}
	})
	ctx := context.Background()

	_, err := m.Create(ctx, "a", "1.1.1.1")
	require.NoError(t, err)
	_, err = m.Create(ctx, "b", "1.1.1.1")
	require.NoError(t, err)
	_, err = m.Create(ctx, "c", "1.1.1.1")
	assert.ErrorIs(t, err, ErrRateLimited)
	_, err = m.Create(ctx, "a", "1.1.1.1")
	assert.NoError(t, err, "re-sharing an existing diagram stores nothing and is not limited")

	_, err = m.Create(ctx, "c", "2.2.2.2")
	assert.NoError(t, err, "limit is per client IP")

	clock.advance(time.Hour)
	_, err = m.Create(ctx, "c", "1.1.1.1")
	assert.NoError(t, err, "limit resets after the window")
}

func TestCleanupByLastView(t *testing.T) {
	m, clock := newTestManager(t, func(c *Config) { c.Retention = util.Duration(30 * day) })
	ctx := context.Background()

	viewed, err := m.Create(ctx, "viewed", "ip")
	require.NoError(t, err)
	idle, err := m.Create(ctx, "idle", "ip")
	require.NoError(t, err)

	clock.advance(20 * day)
	_, err = m.Get(ctx, viewed.Id)
	require.NoError(t, err)

	clock.advance(15 * day) // idle: 35 days since last view, viewed: 15 days
	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	_, err = m.Find(ctx, idle.Id)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = m.Find(ctx, viewed.Id)
	assert.NoError(t, err)
}

func TestCleanupKeepsPinnedAndExtended(t *testing.T) {
	m, clock := newTestManager(t, func(c *Config) { c.Retention = util.Duration(30 * day) })
	ctx := context.Background()

	pinned, _ := m.Create(ctx, "pinned", "ip")
	extended, _ := m.Create(ctx, "extended", "ip")
	require.NoError(t, m.SetPinned(ctx, pinned.Id, true))
	_, err := m.Extend(ctx, extended.Id, 100*day)
	require.NoError(t, err)

	clock.advance(60 * day)
	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n)

	d, err := m.Find(ctx, extended.Id)
	require.NoError(t, err)
	e, err := m.Expiry(ctx, d)
	require.NoError(t, err)
	require.NotNil(t, e.KeepUntil, "extension outlasts the idle window, so it is reported")
	assert.Equal(t, 30, e.IdleDays)

	clock.advance(41 * day) // past the extension and past the idle window
	n, err = m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	_, err = m.Find(ctx, pinned.Id)
	assert.NoError(t, err)
}

func TestRetentionZeroNeverExpires(t *testing.T) {
	m, clock := newTestManager(t, func(c *Config) { c.Retention = 0 })
	ctx := context.Background()

	d, _ := m.Create(ctx, "forever", "ip")
	clock.advance(10 * 365 * day)
	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n)
	_, ok := m.ExpiresAt(d)
	assert.False(t, ok)
}

func TestMigrateBackfillsLastView(t *testing.T) {
	m, clock := newTestManager(t, nil)
	ctx := context.Background()

	// A row saved before view tracking existed.
	require.NoError(t, m.db.Exec("INSERT INTO diagrams (id, source, created_at) VALUES (?, ?, ?)",
		"legacyxid", "A", clock.t.Add(-365*day)).Error)
	require.NoError(t, m.Migrate(ctx))

	d, err := m.Find(ctx, "legacyxid")
	require.NoError(t, err)
	assert.True(t, d.LastViewedAt.Equal(clock.t))
}

func TestDisabled(t *testing.T) {
	m, _ := newTestManager(t, func(c *Config) { c.Enabled = false })
	ctx := context.Background()

	_, err := m.Create(ctx, "A", "ip")
	assert.ErrorIs(t, err, ErrStorageDisabled)
	_, err = m.Get(ctx, "whatever")
	assert.ErrorIs(t, err, ErrStorageDisabled)
}

func sqliteMemory() gorm.Dialector { return sqlite.Open(":memory:") }
