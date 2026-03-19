package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Plan defines the feature set and limits available to a tenant.
type Plan struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nombre          string          `gorm:"type:varchar(100);not null"`
	MaxTerminales   int             `gorm:"not null;default:1"`
	MaxProductos    int             `gorm:"not null;default:0"` // 0 = sin límite
	PrecioMensual   decimal.Decimal `gorm:"type:decimal(10,2);not null;default:0"`
	Features        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	Activo          bool            `gorm:"not null;default:true"`
	CreatedAt       time.Time
}

func (Plan) TableName() string { return "plans" }

// Tenant represents a single commercial account (one kiosk / one business).
// A tenant maps 1-to-1 with a paying subscriber.
type Tenant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Slug      string    `gorm:"type:varchar(63);uniqueIndex;not null"`
	Nombre    string    `gorm:"type:varchar(255);not null"`
	PlanID    *uuid.UUID `gorm:"type:uuid"`
	CUIT        *string `gorm:"column:cuit;type:varchar(13)"`
	TipoNegocio string  `gorm:"type:varchar(30);not null;default:'kiosco'"`
	Activo      bool    `gorm:"not null;default:true"`
	CreatedAt time.Time

	Plan *Plan `gorm:"foreignKey:PlanID"`
}

func (Tenant) TableName() string { return "tenants" }
