package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base replaces gorm.Model to use UUID primary keys instead of uint.
// IDs are assigned in Go by BeforeCreate (not by a DB-level default) so the
// schema stays portable — SQLite test databases can't parse
// default:gen_random_uuid(), and the hook fires on every GORM insert anyway.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                json:"deleted_at,omitempty"`
}

// BeforeCreate ensures a UUID is assigned if none was set programmatically.
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
