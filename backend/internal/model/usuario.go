package model

import (
	"time"

	"github.com/google/uuid"
)

// Usuario stores system users with role-based access.
// Rol: "cajero" | "supervisor" | "administrador"
type Usuario struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Username string    `gorm:"uniqueIndex;not null"`
	Nombre   string    `gorm:"not null"`
	Email    *string
	PasswordHash string `gorm:"not null"`
	Rol          string `gorm:"type:varchar(20);not null"`
	// PuntoDeVenta restricts a cashier to a specific register; nil = all registers
	PuntoDeVenta *int
	// DeviceID identifies the physical POS terminal this user is registered on.
	// Generated once by the PWA and persisted in localStorage.
	DeviceID           *string `gorm:"type:varchar(36)"`
	Activo             bool    `gorm:"not null;default:true"`
	MustChangePassword bool    `gorm:"not null;default:false"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
