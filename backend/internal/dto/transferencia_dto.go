package dto

// ─── Stock Sucursal DTOs ────────────────────────────────────────────────────

type StockSucursalResponse struct {
	ID          string `json:"id"`
	ProductoID  string `json:"producto_id"`
	Producto    string `json:"producto"`
	SucursalID  string `json:"sucursal_id"`
	StockActual int    `json:"stock_actual"`
	StockMinimo int    `json:"stock_minimo"`
	UpdatedAt   string `json:"updated_at"`
}

type StockSucursalListResponse struct {
	Data  []StockSucursalResponse `json:"data"`
	Total int64                   `json:"total"`
}

type AjustarStockSucursalRequest struct {
	ProductoID  string `json:"producto_id"  validate:"required,uuid"`
	SucursalID  string `json:"sucursal_id"  validate:"required,uuid"`
	Delta       int    `json:"delta"        validate:"required,ne=0"`
	Motivo      string `json:"motivo"       validate:"required,min=3,max=500"`
}

// ─── Transferencia DTOs ─────────────────────────────────────────────────────

type TransferenciaItemRequest struct {
	ProductoID string `json:"producto_id" validate:"required,uuid"`
	Cantidad   int    `json:"cantidad"    validate:"required,gt=0"`
}

type CrearTransferenciaRequest struct {
	SucursalOrigenID  string                     `json:"sucursal_origen_id"  validate:"required,uuid"`
	SucursalDestinoID string                     `json:"sucursal_destino_id" validate:"required,uuid"`
	Items             []TransferenciaItemRequest `json:"items"               validate:"required,min=1,dive"`
	Notas             *string                    `json:"notas"               validate:"omitempty,max=1000"`
}

type TransferenciaItemResponse struct {
	ID         string `json:"id"`
	ProductoID string `json:"producto_id"`
	Producto   string `json:"producto"`
	Cantidad   int    `json:"cantidad"`
}

type TransferenciaResponse struct {
	ID                string                      `json:"id"`
	SucursalOrigenID  string                      `json:"sucursal_origen_id"`
	SucursalOrigen    string                      `json:"sucursal_origen"`
	SucursalDestinoID string                      `json:"sucursal_destino_id"`
	SucursalDestino   string                      `json:"sucursal_destino"`
	Estado            string                      `json:"estado"`
	Notas             *string                     `json:"notas,omitempty"`
	CreadoPor         string                      `json:"creado_por"`
	CreadoPorNombre   string                      `json:"creado_por_nombre,omitempty"`
	CompletadoPor     *string                     `json:"completado_por,omitempty"`
	Items             []TransferenciaItemResponse `json:"items"`
	CreatedAt         string                      `json:"created_at"`
	CompletedAt       *string                     `json:"completed_at,omitempty"`
}

type TransferenciaListResponse struct {
	Data  []TransferenciaResponse `json:"data"`
	Total int64                   `json:"total"`
}
