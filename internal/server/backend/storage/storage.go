package storage

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

var ErrStorageDisabled = errors.New("storage is disabled")

type Config struct {
	Enabled bool
}

type Diagram struct {
	Id        string    `json:"id" gorm:"primaryKey"`
	Source    string    `json:"source" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Manager struct {
	db      *gorm.DB
	enabled bool
}

func New(ctx context.Context, conf *Config, db *gorm.DB) (*Manager, error) {
	if conf.Enabled {
		if err := db.AutoMigrate(&Diagram{}); err != nil {
			return nil, err
		}
	}
	return &Manager{db: db, enabled: conf.Enabled}, nil
}

func (m *Manager) Create(ctx context.Context, source string) (*Diagram, error) {
	if !m.enabled {
		return nil, ErrStorageDisabled
	}
	diagram := &Diagram{
		Id:        xid.New().String(),
		Source:    source,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := m.db.WithContext(ctx).Create(diagram).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return diagram, nil
}

func (m *Manager) Get(ctx context.Context, id string) (*Diagram, error) {
	diagram := &Diagram{}
	if err := m.db.WithContext(ctx).First(diagram, "id = ?", id).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return diagram, nil
}

func (m *Manager) Update(ctx context.Context, id string, source string) (*Diagram, error) {
	diagram := &Diagram{}
	if err := m.db.WithContext(ctx).First(diagram, "id = ?", id).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	diagram.Source = source
	diagram.UpdatedAt = time.Now()
	if err := m.db.WithContext(ctx).Save(diagram).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return diagram, nil
}
