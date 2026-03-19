package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Cliente represents a customer with a running credit tab (cuenta corriente / fiado).
type Cliente struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID      uuid.UUID       `gorm:"type:uuid;not null;index"`
	Nombre        string          `gorm:"type:varchar(200);not null"`
	Telefono      *string         `gorm:"type:varchar(50)"`
	Email         *string         `gorm:"type:varchar(200)"`
	DNI           *string         `gorm:"type:varchar(20)"`
	LimiteCredito decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0"`
	SaldoDeudor   decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0"`
	Activo        bool            `gorm:"not null;default:true"`
	Notas         *string         `gorm:"type:text"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Movimientos []MovimientoCuenta `gorm:"foreignKey:ClienteID"`
}

func (Cliente) TableName() string { return "clientes" }

// MovimientoCuenta is an append-only ledger entry for a customer's credit account.
// Tipo: "cargo" (fiado sale), "pago" (payment received), "ajuste" (manual correction).
type MovimientoCuenta struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID        uuid.UUID       `gorm:"type:uuid;not null;index"`
	ClienteID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	Tipo            string          `gorm:"type:varchar(20);not null"`
	Monto           decimal.Decimal `gorm:"type:decimal(12,2);not null"`
	SaldoPosterior  decimal.Decimal `gorm:"type:decimal(12,2);not null"`
	ReferenciaID    *uuid.UUID      `gorm:"type:uuid"`
	ReferenciaTipo  *string         `gorm:"type:varchar(30)"`
	Descripcion     *string         `gorm:"type:text"`
	CreatedAt       time.Time
}

func (MovimientoCuenta) TableName() string { return "movimientos_cuenta" }
