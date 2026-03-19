package model

import (
	"time"

	"github.com/google/uuid"
)

// Sucursal represents a physical branch location within a tenant.
// A tenant can have multiple sucursales. Users, cash sessions, and sales
// can optionally be scoped to a specific sucursal.
type Sucursal struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Nombre    string    `gorm:"type:varchar(200);not null"`
	Direccion *string   `gorm:"type:text"`
	Telefono  *string   `gorm:"type:varchar(50)"`
	Activa     bool      `gorm:"not null;default:true"`
	EsDeposito bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Sucursal) TableName() string { return "sucursales" }
