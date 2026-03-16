package model

import (
	"time"

	"github.com/google/uuid"
)

// Categoria represents a product category used to classify products.
type Categoria struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID uuid.UUID `gorm:"type:uuid;not null;index"`
	// Nombre is unique per tenant (uq_categorias_tenant_nombre), not globally.
	Nombre   string    `gorm:"index;not null"`
	Descripcion *string
	Activo      bool `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName overrides GORM's default singular → plural logic for Spanish names.
func (Categoria) TableName() string { return "categorias" }
