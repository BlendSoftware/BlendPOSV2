package dto

// ─── Request DTOs ────────────────────────────────────────────────────────────

type CrearSucursalRequest struct {
	Nombre    string  `json:"nombre"    validate:"required,min=2,max=200"`
	Direccion *string `json:"direccion" validate:"omitempty"`
	Telefono  *string `json:"telefono"  validate:"omitempty,max=50"`
}

type UpdateSucursalRequest struct {
	Nombre    *string `json:"nombre"    validate:"omitempty,min=2,max=200"`
	Direccion *string `json:"direccion" validate:"omitempty"`
	Telefono  *string `json:"telefono"  validate:"omitempty,max=50"`
	Activa    *bool   `json:"activa"    validate:"omitempty"`
}

// ─── Response DTOs ───────────────────────────────────────────────────────────

type SucursalResponse struct {
	ID        string  `json:"id"`
	Nombre    string  `json:"nombre"`
	Direccion *string `json:"direccion,omitempty"`
	Telefono  *string `json:"telefono,omitempty"`
	Activa    bool    `json:"activa"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type SucursalListResponse struct {
	Data  []SucursalResponse `json:"data"`
	Total int64              `json:"total"`
}
