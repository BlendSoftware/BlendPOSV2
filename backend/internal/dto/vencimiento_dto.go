package dto

// ─── Request DTOs ────────────────────────────────────────────────────────────

type CrearLoteRequest struct {
	ProductoID       string  `json:"producto_id"       validate:"required,uuid"`
	CodigoLote       *string `json:"codigo_lote"       validate:"omitempty,max=100"`
	FechaVencimiento string  `json:"fecha_vencimiento" validate:"required"` // YYYY-MM-DD
	Cantidad         int     `json:"cantidad"          validate:"required,min=1"`
}

// ─── Response DTOs ───────────────────────────────────────────────────────────

type LoteResponse struct {
	ID               string  `json:"id"`
	ProductoID       string  `json:"producto_id"`
	ProductoNombre   string  `json:"producto_nombre"`
	CodigoLote       *string `json:"codigo_lote"`
	FechaVencimiento string  `json:"fecha_vencimiento"` // YYYY-MM-DD
	Cantidad         int     `json:"cantidad"`
	CreatedAt        string  `json:"created_at"`
}

type AlertaVencimientoResponse struct {
	ID               string  `json:"id"`
	ProductoID       string  `json:"producto_id"`
	ProductoNombre   string  `json:"producto_nombre"`
	CodigoLote       *string `json:"codigo_lote"`
	FechaVencimiento string  `json:"fecha_vencimiento"` // YYYY-MM-DD
	DiasRestantes    int     `json:"dias_restantes"`
	Cantidad         int     `json:"cantidad"`
	Estado           string  `json:"estado"` // "vencido" | "critico" | "proximo"
}
