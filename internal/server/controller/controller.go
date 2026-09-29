package controller

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/bluemir/grim/internal/server/backend"
)

const (
	// storageCleanupInterval is how often expired diagrams are swept.
	storageCleanupInterval = time.Hour
	// storageReminderInterval is how often due confirmation mails are sent.
	storageReminderInterval = time.Hour
)

// Controller watches job lifecycle events and performs post-processing.
type Controller struct {
	ctx      context.Context
	backends *backend.Backends
}

// New creates a Controller. Call Run() to start the event loop.
func New(ctx context.Context, bs *backend.Backends) *Controller {
	return &Controller{ctx: ctx, backends: bs}
}

// Run starts the event loop. It blocks until the context is cancelled.
func (c *Controller) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	//eg.Go(c.handleJob(ctx))
	eg.Go(c.cleanupStorage(ctx))
	eg.Go(c.sendStorageReminders(ctx))

	return eg.Wait()
}

// cleanupStorage deletes expired diagrams, once at startup and then periodically.
func (c *Controller) cleanupStorage(ctx context.Context) func() error {
	return func() error {
		if !c.backends.Storage.Enabled() {
			return nil
		}
		ticker := time.NewTicker(storageCleanupInterval)
		defer ticker.Stop()
		for {
			n, err := c.backends.Storage.Cleanup(ctx)
			if err != nil {
				logrus.Warnf("storage cleanup failed: %v", err)
			} else if n > 0 {
				logrus.Infof("storage cleanup: deleted %d expired diagrams", n)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}
}

// sendStorageReminders mails due confirmation reminders. It runs apart from
// cleanup so a slow or unreachable SMTP server never delays deletion.
func (c *Controller) sendStorageReminders(ctx context.Context) func() error {
	return func() error {
		if !c.backends.Storage.VerificationEnabled() {
			return nil
		}
		ticker := time.NewTicker(storageReminderInterval)
		defer ticker.Stop()
		for {
			if err := c.backends.Storage.SendReminders(ctx); err != nil {
				logrus.Warnf("storage reminders failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}
}
