package model

import (
	"time"

	"github.com/google/uuid"
)

// StockSucursal tracks per-branch stock levels for a product.
// This is an ADDITIONAL layer on top of productos.stock_actual (global stock).
// Tenants without sucursales configured continue using the global stock as before.
type StockSucursal struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductoID  uuid.UUID `gorm:"type:uuid;not null;index"`
	SucursalID  uuid.UUID `gorm:"type:uuid;not null;index"`
	StockActual int       `gorm:"not null;default:0"`
	StockMinimo int       `gorm:"not null;default:5"`
	UpdatedAt   time.Time

	Producto *Producto `gorm:"foreignKey:ProductoID"`
	Sucursal *Sucursal `gorm:"foreignKey:SucursalID"`
}

func (StockSucursal) TableName() string { return "stock_sucursal" }

// TransferenciaStock represents a stock transfer between two branches.
type TransferenciaStock struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID          uuid.UUID `gorm:"type:uuid;not null;index"`
	SucursalOrigenID  uuid.UUID `gorm:"type:uuid;not null"`
	SucursalDestinoID uuid.UUID `gorm:"type:uuid;not null"`
	Estado            string    `gorm:"type:varchar(20);not null;default:'pendiente'"`
	Notas             *string   `gorm:"type:text"`
	CreadoPor         uuid.UUID `gorm:"type:uuid;not null"`
	CompletadoPor     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt         time.Time
	CompletedAt       *time.Time

	Items           []TransferenciaItem `gorm:"foreignKey:TransferenciaID"`
	SucursalOrigen  *Sucursal           `gorm:"foreignKey:SucursalOrigenID"`
	SucursalDestino *Sucursal           `gorm:"foreignKey:SucursalDestinoID"`
	Creador         *Usuario            `gorm:"foreignKey:CreadoPor"`
}

func (TransferenciaStock) TableName() string { return "transferencias_stock" }

// TransferenciaItem is a line item within a stock transfer.
type TransferenciaItem struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransferenciaID uuid.UUID `gorm:"type:uuid;not null"`
	ProductoID      uuid.UUID `gorm:"type:uuid;not null"`
	Cantidad        int       `gorm:"not null"`

	Producto *Producto `gorm:"foreignKey:ProductoID"`
}

func (TransferenciaItem) TableName() string { return "transferencia_items" }
