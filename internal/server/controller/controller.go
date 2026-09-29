package controller

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/bluemir/grim/internal/server/backend"
)

// storageCleanupInterval is how often expired diagrams are swept.
const storageCleanupInterval = time.Hour

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

	return eg.Wait()
}

// cleanupStorage deletes expired diagrams once at startup and then periodically.
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
