package dto

import "github.com/shopspring/decimal"

// ── Request DTOs ──────────────────────────────────────────────────────────────

// RegisterTenantRequest is the payload for POST /v1/public/register.
type RegisterTenantRequest struct {
	// Tenant fields
	NombreNegocio string `json:"nombre_negocio" validate:"required,min=2,max=255"`
	Slug          string `json:"slug"           validate:"required,min=2,max=63,alphanum"`

	// Initial admin user
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8"`
	Nombre   string `json:"nombre"   validate:"required,min=2,max=100"`
	Email    string `json:"email"    validate:"omitempty,email"`
}

type ActualizarTenantRequest struct {
	Nombre string  `json:"nombre" validate:"omitempty,min=2,max=255"`
	CUIT   *string `json:"cuit"   validate:"omitempty,len=11"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type PlanResponse struct {
	ID            string          `json:"id"`
	Nombre        string          `json:"nombre"`
	MaxTerminales int             `json:"max_terminales"`
	MaxProductos  int             `json:"max_productos"`
	PrecioMensual decimal.Decimal `json:"precio_mensual"`
}

type TenantResponse struct {
	ID        string          `json:"id"`
	Slug      string          `json:"slug"`
	Nombre    string          `json:"nombre"`
	CUIT      *string         `json:"cuit,omitempty"`
	Activo    bool            `json:"activo"`
	Plan      *PlanResponse   `json:"plan,omitempty"`
	CreatedAt string          `json:"created_at"`
}

// RegisterTenantResponse is returned after a successful registration.
// The client receives a ready-to-use JWT so the user is immediately logged in.
type RegisterTenantResponse struct {
	Tenant       TenantResponse `json:"tenant"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	TokenType    string         `json:"token_type"`
	ExpiresIn    int            `json:"expires_in"`
}

// SuperadminTenantListItem is used in the superadmin tenant list.
type SuperadminTenantListItem struct {
	ID           string          `json:"id"`
	Slug         string          `json:"slug"`
	Nombre       string          `json:"nombre"`
	CUIT         *string         `json:"cuit,omitempty"`
	Activo       bool            `json:"activo"`
	Plan         *PlanResponse   `json:"plan,omitempty"`
	TotalVentas  int64           `json:"total_ventas"`
	TotalUsuarios int64          `json:"total_usuarios"`
	CreatedAt    string          `json:"created_at"`
}

type SuperadminMetricsResponse struct {
	TotalTenants  int64 `json:"total_tenants"`
	TenantActivos int64 `json:"tenants_activos"`
}

type CambiarPlanRequest struct {
	PlanID string `json:"plan_id" validate:"required,uuid"`
}

type ToggleTenantRequest struct {
	Activo bool `json:"activo"`
}
