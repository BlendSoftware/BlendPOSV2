package dto

import "github.com/shopspring/decimal"

// ─── Request DTOs ────────────────────────────────────────────────────────────

type CrearClienteRequest struct {
	Nombre        string          `json:"nombre"         validate:"required,min=2,max=200"`
	Telefono      *string         `json:"telefono"       validate:"omitempty,max=50"`
	Email         *string         `json:"email"          validate:"omitempty,email,max=200"`
	DNI           *string         `json:"dni"            validate:"omitempty,max=20"`
	LimiteCredito decimal.Decimal `json:"limite_credito" validate:"min=0"`
	Notas         *string         `json:"notas"          validate:"omitempty"`
}

type UpdateClienteRequest struct {
	Nombre        *string          `json:"nombre"         validate:"omitempty,min=2,max=200"`
	Telefono      *string          `json:"telefono"       validate:"omitempty,max=50"`
	Email         *string          `json:"email"          validate:"omitempty,email,max=200"`
	DNI           *string          `json:"dni"            validate:"omitempty,max=20"`
	LimiteCredito *decimal.Decimal `json:"limite_credito" validate:"omitempty,min=0"`
	Activo        *bool            `json:"activo"         validate:"omitempty"`
	Notas         *string          `json:"notas"          validate:"omitempty"`
}

type RegistrarPagoClienteRequest struct {
	Monto       decimal.Decimal `json:"monto"       validate:"required,gt=0"`
	Descripcion *string         `json:"descripcion" validate:"omitempty,max=500"`
}

// ─── Response DTOs ───────────────────────────────────────────────────────────

type ClienteResponse struct {
	ID            string          `json:"id"`
	Nombre        string          `json:"nombre"`
	Telefono      *string         `json:"telefono,omitempty"`
	Email         *string         `json:"email,omitempty"`
	DNI           *string         `json:"dni,omitempty"`
	LimiteCredito decimal.Decimal `json:"limite_credito"`
	SaldoDeudor   decimal.Decimal `json:"saldo_deudor"`
	CreditoDisponible decimal.Decimal `json:"credito_disponible"`
	Activo        bool            `json:"activo"`
	Notas         *string         `json:"notas,omitempty"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

type ClienteListResponse struct {
	Data  []ClienteResponse `json:"data"`
	Total int64             `json:"total"`
}

type MovimientoCuentaResponse struct {
	ID             string          `json:"id"`
	ClienteID      string          `json:"cliente_id"`
	Tipo           string          `json:"tipo"`
	Monto          decimal.Decimal `json:"monto"`
	SaldoPosterior decimal.Decimal `json:"saldo_posterior"`
	ReferenciaID   *string         `json:"referencia_id,omitempty"`
	ReferenciaTipo *string         `json:"referencia_tipo,omitempty"`
	Descripcion    *string         `json:"descripcion,omitempty"`
	CreatedAt      string          `json:"created_at"`
}

type MovimientosListResponse struct {
	Data  []MovimientoCuentaResponse `json:"data"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Limit int                        `json:"limit"`
}

type DeudorResponse struct {
	ID          string          `json:"id"`
	Nombre      string          `json:"nombre"`
	Telefono    *string         `json:"telefono,omitempty"`
	SaldoDeudor decimal.Decimal `json:"saldo_deudor"`
	LimiteCredito decimal.Decimal `json:"limite_credito"`
}

type ListDeudoresResponse struct {
	Data  []DeudorResponse `json:"data"`
	Total int64            `json:"total"`
}
