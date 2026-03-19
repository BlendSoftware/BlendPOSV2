package dto

import "github.com/shopspring/decimal"

// ─── Request DTOs ────────────────────────────────────────────────────────────

type CrearProductoRequest struct {
	CodigoBarras string          `json:"codigo_barras" validate:"required,min=8,max=18"`
	Nombre       string          `json:"nombre"        validate:"required,min=2,max=120"`
	Descripcion  *string         `json:"descripcion"`
	Categoria    string          `json:"categoria"     validate:"required"`
	PrecioCosto  decimal.Decimal `json:"precio_costo"  validate:"required"`
	PrecioVenta  decimal.Decimal `json:"precio_venta"  validate:"required"`
	StockActual  int             `json:"stock_actual"  validate:"min=0"`
	StockMinimo  int             `json:"stock_minimo"  validate:"min=0"`
	UnidadMedida string          `json:"unidad_medida"`
	ProveedorID  *string         `json:"proveedor_id"  validate:"omitempty,uuid"`
	EsPadre      bool            `json:"es_padre"`
}

type ActualizarProductoRequest struct {
	Nombre              *string          `json:"nombre"                validate:"omitempty,min=2,max=120"`
	Descripcion         *string          `json:"descripcion"`
	Categoria           *string          `json:"categoria"`
	PrecioCosto         *decimal.Decimal `json:"precio_costo"`
	PrecioVenta         *decimal.Decimal `json:"precio_venta"`
	StockMinimo         *int             `json:"stock_minimo"          validate:"omitempty,min=0"`
	UnidadMedida        *string          `json:"unidad_medida"`
	ProveedorID         *string          `json:"proveedor_id"          validate:"omitempty,uuid"`
	ControlaVencimiento *bool            `json:"controla_vencimiento"`
	EsPadre             *bool            `json:"es_padre"`
}

// ─── Filter / Pagination ─────────────────────────────────────────────────────

type ProductoFilter struct {
	Barcode     string `form:"barcode"`
	Nombre      string `form:"nombre"`
	Categoria   string `form:"categoria"`
	ProveedorID string `form:"proveedor_id"`
	// Activo: "true" = solo activos (default), "false" = solo inactivos, "all" = todos
	Activo       string `form:"activo,default=true"`
	// IncluirVariantes: "true" = include child variants in list, default is "false" (only root/parent products)
	IncluirVariantes string `form:"incluir_variantes,default=false"`
	// UpdatedAfter: ISO-8601 timestamp — return only products updated after this time.
	// Used by the frontend delta-sync to avoid downloading the full catalog on every POS mount.
	UpdatedAfter string `form:"updated_after"`
	Page         int    `form:"page,default=1"  validate:"min=1"`
	Limit        int    `form:"limit,default=20" validate:"min=1,max=100"`
}

// ─── Response DTOs ───────────────────────────────────────────────────────────

type ProductoResponse struct {
	ID           string          `json:"id"`
	CodigoBarras string          `json:"codigo_barras"`
	Nombre       string          `json:"nombre"`
	Descripcion  *string         `json:"descripcion"`
	Categoria    string          `json:"categoria"`
	PrecioCosto  decimal.Decimal `json:"precio_costo"`
	PrecioVenta  decimal.Decimal `json:"precio_venta"`
	MargenPct    decimal.Decimal `json:"margen_pct"`
	StockActual  int             `json:"stock_actual"`
	StockMinimo  int             `json:"stock_minimo"`
	UnidadMedida string          `json:"unidad_medida"`
	EsPadre             bool              `json:"es_padre"`
	PadreID             *string           `json:"padre_id,omitempty"`
	VarianteAtributos   map[string]string `json:"variante_atributos,omitempty"`
	VarianteNombre      *string           `json:"variante_nombre,omitempty"`
	Activo              bool              `json:"activo"`
	ControlaVencimiento bool              `json:"controla_vencimiento"`
	ProveedorID         *string           `json:"proveedor_id"`
	CantidadVariantes   int               `json:"cantidad_variantes,omitempty"`
}

type ProductoListResponse struct {
	Data       []ProductoResponse `json:"data"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// ConsultaPreciosResponse is returned by the public price check endpoint (no auth required).
type ConsultaPreciosResponse struct {
	Nombre          string          `json:"nombre"`
	PrecioVenta     decimal.Decimal `json:"precio_venta"`
	StockDisponible int             `json:"stock_disponible"`
	Categoria       string          `json:"categoria"`
	Promocion       *string         `json:"promocion"`
}

// AjustarStockRequest is used by PATCH /v1/productos/:id/stock (RF-09 / manual adjustment).
type AjustarStockRequest struct {
	Delta  int    `json:"delta"  validate:"required,ne=0"`
	Motivo string `json:"motivo" validate:"required,min=3"`
}

// ─── Variant DTOs ───────────────────────────────────────────────────────────

// CrearVarianteRequest creates a variant (child) product from a parent product.
type CrearVarianteRequest struct {
	Atributos    map[string]string `json:"atributos"     validate:"required,min=1"`
	CodigoBarras string            `json:"codigo_barras"  validate:"required,min=8,max=18"`
	PrecioVenta  *decimal.Decimal  `json:"precio_venta"`
	PrecioCosto  *decimal.Decimal  `json:"precio_costo"`
	StockActual  int               `json:"stock_actual"   validate:"min=0"`
}

// ─── Bulk Import DTOs ───────────────────────────────────────────────────────

// BulkCrearProductosRequest wraps an array of products for POST /v1/productos/bulk.
type BulkCrearProductosRequest struct {
	Productos []CrearProductoRequest `json:"productos" validate:"required,min=1,max=500,dive"`
}

// BulkImportResult represents the outcome of a single product in a bulk import.
type BulkImportResult struct {
	Index   int    `json:"index"`
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// BulkImportResponse is the response for POST /v1/productos/bulk.
type BulkImportResponse struct {
	Total   int                `json:"total"`
	Created int                `json:"created"`
	Failed  int                `json:"failed"`
	Results []BulkImportResult `json:"results"`
}
