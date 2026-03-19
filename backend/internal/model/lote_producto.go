package model

import (
	"time"

	"github.com/google/uuid"
)

// LoteProducto represents a batch/lot of a product with an expiry date.
// A single product can have multiple lots with different expiry dates
// (e.g., milk received on different dates).
type LoteProducto struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID          uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductoID        uuid.UUID `gorm:"type:uuid;not null;index"`
	CodigoLote        *string   `gorm:"type:varchar(100)"`
	FechaVencimiento  time.Time `gorm:"type:date;not null"`
	Cantidad          int       `gorm:"not null;default:0"`
	CreatedAt         time.Time

	Producto *Producto `gorm:"foreignKey:ProductoID"`
}
