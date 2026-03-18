package backend

import (
	"context"

	"github.com/bluemir/grim/internal/pubsub"
	"github.com/bluemir/grim/internal/server/backend/auth"
	"github.com/bluemir/grim/internal/server/backend/posts"
	"github.com/bluemir/grim/internal/server/backend/storage"
	"gorm.io/gorm"
)

// Config from file
type Config struct {
	Auth struct {
		Salt string
	}
	Posts   posts.Config
	Storage storage.Config
}
type Backends struct {
	Auth    *auth.Manager
	Events  *pubsub.Hub
	Posts   *posts.Manager
	Storage *storage.Manager
}

func Initialize(ctx context.Context, conf *Config, db *gorm.DB) (*Backends, error) {
	events, err := pubsub.NewHub(ctx)
	if err != nil {
		return nil, err
	}
	// init components

	authManager, err := auth.New(db, conf.Auth.Salt)
	if err != nil {
		return nil, err
	}
	postManager, err := posts.New(ctx, &conf.Posts, db, events)
	if err != nil {
		return nil, err
	}
	storageManager, err := storage.New(ctx, &conf.Storage, db)
	if err != nil {
		return nil, err
	}

	return &Backends{
		Events:  events,
		Auth:    authManager,
		Posts:   postManager,
		Storage: storageManager,
	}, nil
}
