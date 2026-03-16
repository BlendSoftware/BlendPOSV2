package model

import (
	"time"

	"github.com/google/uuid"
)

// Proveedor represents a supplier with commercial data.
type Proveedor struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null;index"`
	RazonSocial string    `gorm:"not null"`
	// CUIT is unique per tenant (uq_proveedores_tenant_cuit), not globally.
	CUIT        string    `gorm:"column:cuit;index;not null"`
	Telefono      *string
	Email         *string
	Direccion     *string
	CondicionPago *string
	Activo        bool `gorm:"not null;default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Productos []Producto          `gorm:"foreignKey:ProveedorID"`
	Contactos []ContactoProveedor `gorm:"foreignKey:ProveedorID"`
}

func (Proveedor) TableName() string { return "proveedores" }
