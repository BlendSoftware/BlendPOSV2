package dto

import "github.com/shopspring/decimal"

// ── Request DTOs ──────────────────────────────────────────────────────────────

// RegisterTenantRequest is the payload for POST /v1/public/register.
type RegisterTenantRequest struct {
	// Tenant fields
	NombreNegocio string `json:"nombre_negocio" validate:"required,min=2,max=255"`
	Slug          string `json:"slug"           validate:"omitempty,min=2,max=63"` // optional — auto-generated from nombre_negocio
	CUIT          string `json:"cuit"           validate:"omitempty,len=11"`       // optional

	// Business type preset (optional — defaults to "kiosco")
	TipoNegocio string `json:"tipo_negocio" validate:"omitempty,oneof=kiosco carniceria minimarket verduleria"`

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
	ID            string            `json:"id"`
	Nombre        string            `json:"nombre"`
	MaxTerminales int               `json:"max_terminales"`
	MaxProductos  int               `json:"max_productos"`
	PrecioMensual decimal.Decimal   `json:"precio_mensual"`
	Features      map[string]bool   `json:"features"`
}

type TenantResponse struct {
	ID          string        `json:"id"`
	Slug        string        `json:"slug"`
	Nombre      string        `json:"nombre"`
	CUIT        *string       `json:"cuit,omitempty"`
	TipoNegocio string       `json:"tipo_negocio"`
	Activo      bool          `json:"activo"`
	Plan        *PlanResponse `json:"plan,omitempty"`
	CreatedAt   string        `json:"created_at"`
}

// RegisterTenantResponse is returned after a successful registration.
// The client receives a ready-to-use JWT so the user is immediately logged in.
type RegisterTenantResponse struct {
	Tenant       TenantResponse `json:"tenant"`
	User         UsuarioResponse `json:"user"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	TokenType    string         `json:"token_type"`
	ExpiresIn    int            `json:"expires_in"`
}

// ── Superadmin Request DTOs ───────────────────────────────────────────────────

// TenantListRequest holds pagination and filter params for GET /v1/superadmin/tenants.
type TenantListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`  // buscar por nombre o slug
	Status   string `form:"status"`  // "active", "inactive", "all" (default "all")
	PlanID   string `form:"plan_id"` // filtrar por plan UUID
}

// Defaults applies sane defaults for pagination.
func (r *TenantListRequest) Defaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	if r.Status == "" {
		r.Status = "all"
	}
}

// ── Superadmin Response DTOs ─────────────────────────────────────────────────

// SuperadminTenantListItem is used in the superadmin tenant list.
type SuperadminTenantListItem struct {
	ID             string        `json:"id"`
	Slug           string        `json:"slug"`
	Nombre         string        `json:"nombre"`
	CUIT           *string       `json:"cuit,omitempty"`
	Activo         bool          `json:"activo"`
	Plan           *PlanResponse `json:"plan,omitempty"`
	TotalVentas    int64         `json:"total_ventas"`
	TotalProductos int64         `json:"total_productos"`
	TotalUsuarios  int64         `json:"total_usuarios"`
	UltimaVenta    string        `json:"ultima_venta,omitempty"`
	CreatedAt      string        `json:"created_at"`
}

// TenantListResponse is the paginated response for GET /v1/superadmin/tenants.
type TenantListResponse struct {
	Tenants    []SuperadminTenantListItem `json:"tenants"`
	Total      int64                      `json:"total"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"page_size"`
	TotalPages int                        `json:"total_pages"`
}

// PlanCountDTO maps a plan name to tenant count.
type PlanCountDTO struct {
	PlanNombre string `json:"plan_nombre"`
	Count      int64  `json:"count"`
}

type SuperadminMetricsResponse struct {
	TotalTenants    int64          `json:"total_tenants"`
	TenantActivos   int64          `json:"tenants_activos"`
	TotalVentas     int64          `json:"total_ventas"`
	VentasUltimoMes int64          `json:"ventas_ultimo_mes"`
	TenantsPorPlan  []PlanCountDTO `json:"tenants_por_plan"`
}

type CambiarPlanRequest struct {
	PlanID string `json:"plan_id" validate:"required,uuid"`
}

type ToggleTenantRequest struct {
	Activo bool `json:"activo"`
}

// ── Preset DTOs ─────────────────────────────────────────────────────────────

// PresetCategoryResponse describes a category in a business type preset.
type PresetCategoryResponse struct {
	Nombre        string `json:"nombre"`
	ProductCount  int    `json:"product_count"`
}

// PresetResponse describes the preset data for a business type.
type PresetResponse struct {
	TipoNegocio   string                   `json:"tipo_negocio"`
	Label         string                   `json:"label"`
	TotalCategorias int                    `json:"total_categorias"`
	TotalProductos  int                    `json:"total_productos"`
	Categorias    []PresetCategoryResponse `json:"categorias"`
}
