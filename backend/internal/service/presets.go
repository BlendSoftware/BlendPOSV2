package service

import (
	"context"
	"fmt"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/tenantctx"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ── Preset data structures ──────────────────────────────────────────────────

type presetProduct struct {
	Nombre       string
	CodigoBarras string
	PrecioCosto  string
	PrecioVenta  string
	UnidadMedida string
	StockActual  int
	Categoria    string // matches category name in the same preset
}

type presetCategory struct {
	Nombre string
}

type businessPreset struct {
	Label      string
	Categorias []presetCategory
	Productos  []presetProduct
}

// ── Hardcoded presets ───────────────────────────────────────────────────────

var presets = map[string]businessPreset{
	"kiosco": {
		Label: "Kiosco",
		Categorias: []presetCategory{
			{Nombre: "Golosinas"},
			{Nombre: "Bebidas"},
			{Nombre: "Cigarrillos"},
			{Nombre: "Lácteos"},
			{Nombre: "Limpieza"},
			{Nombre: "Fiambres"},
			{Nombre: "Snacks"},
			{Nombre: "Panadería"},
		},
		Productos: []presetProduct{
			{Nombre: "Alfajor Havanna", CodigoBarras: "7790001000101", PrecioCosto: "500", PrecioVenta: "800", UnidadMedida: "unidad", StockActual: 20, Categoria: "Golosinas"},
			{Nombre: "Coca-Cola 500ml", CodigoBarras: "7790001000102", PrecioCosto: "750", PrecioVenta: "1200", UnidadMedida: "unidad", StockActual: 24, Categoria: "Bebidas"},
			{Nombre: "Marlboro Box", CodigoBarras: "7790001000103", PrecioCosto: "2200", PrecioVenta: "3000", UnidadMedida: "unidad", StockActual: 10, Categoria: "Cigarrillos"},
			{Nombre: "Leche La Serenísima 1L", CodigoBarras: "7790001000104", PrecioCosto: "800", PrecioVenta: "1100", UnidadMedida: "unidad", StockActual: 12, Categoria: "Lácteos"},
			{Nombre: "Lavandina Ayudín 1L", CodigoBarras: "7790001000105", PrecioCosto: "600", PrecioVenta: "900", UnidadMedida: "unidad", StockActual: 6, Categoria: "Limpieza"},
			{Nombre: "Queso Cremoso kg", CodigoBarras: "7790001000106", PrecioCosto: "5500", PrecioVenta: "7500", UnidadMedida: "kg", StockActual: 5, Categoria: "Fiambres"},
			{Nombre: "Papas Lays 150g", CodigoBarras: "7790001000107", PrecioCosto: "900", PrecioVenta: "1400", UnidadMedida: "unidad", StockActual: 15, Categoria: "Snacks"},
			{Nombre: "Medialunas x6", CodigoBarras: "7790001000108", PrecioCosto: "1200", PrecioVenta: "1800", UnidadMedida: "unidad", StockActual: 8, Categoria: "Panadería"},
		},
	},
	"carniceria": {
		Label: "Carnicería",
		Categorias: []presetCategory{
			{Nombre: "Vacuno"},
			{Nombre: "Cerdo"},
			{Nombre: "Pollo"},
			{Nombre: "Achuras"},
			{Nombre: "Embutidos"},
			{Nombre: "Congelados"},
		},
		Productos: []presetProduct{
			{Nombre: "Asado kg", CodigoBarras: "7790002000101", PrecioCosto: "6500", PrecioVenta: "8500", UnidadMedida: "kg", StockActual: 30, Categoria: "Vacuno"},
			{Nombre: "Vacío kg", CodigoBarras: "7790002000102", PrecioCosto: "7500", PrecioVenta: "9800", UnidadMedida: "kg", StockActual: 20, Categoria: "Vacuno"},
			{Nombre: "Nalga kg", CodigoBarras: "7790002000103", PrecioCosto: "6800", PrecioVenta: "8900", UnidadMedida: "kg", StockActual: 15, Categoria: "Vacuno"},
			{Nombre: "Bondiola kg", CodigoBarras: "7790002000104", PrecioCosto: "5500", PrecioVenta: "7200", UnidadMedida: "kg", StockActual: 10, Categoria: "Cerdo"},
			{Nombre: "Pechuga kg", CodigoBarras: "7790002000105", PrecioCosto: "4000", PrecioVenta: "5500", UnidadMedida: "kg", StockActual: 15, Categoria: "Pollo"},
			{Nombre: "Chorizo kg", CodigoBarras: "7790002000106", PrecioCosto: "4500", PrecioVenta: "6000", UnidadMedida: "kg", StockActual: 10, Categoria: "Embutidos"},
			{Nombre: "Morcilla kg", CodigoBarras: "7790002000107", PrecioCosto: "3500", PrecioVenta: "4800", UnidadMedida: "kg", StockActual: 8, Categoria: "Embutidos"},
			{Nombre: "Milanesas kg", CodigoBarras: "7790002000108", PrecioCosto: "5000", PrecioVenta: "7000", UnidadMedida: "kg", StockActual: 12, Categoria: "Congelados"},
		},
	},
	"minimarket": {
		Label: "Minimarket",
		Categorias: []presetCategory{
			{Nombre: "Almacén"},
			{Nombre: "Bebidas"},
			{Nombre: "Lácteos"},
			{Nombre: "Limpieza"},
			{Nombre: "Higiene"},
			{Nombre: "Golosinas"},
			{Nombre: "Congelados"},
			{Nombre: "Frutas y Verduras"},
		},
		Productos: []presetProduct{
			{Nombre: "Arroz 1kg", CodigoBarras: "7790003000101", PrecioCosto: "800", PrecioVenta: "1200", UnidadMedida: "unidad", StockActual: 30, Categoria: "Almacén"},
			{Nombre: "Fideos 500g", CodigoBarras: "7790003000102", PrecioCosto: "500", PrecioVenta: "800", UnidadMedida: "unidad", StockActual: 40, Categoria: "Almacén"},
			{Nombre: "Aceite Girasol 1L", CodigoBarras: "7790003000103", PrecioCosto: "1200", PrecioVenta: "1800", UnidadMedida: "unidad", StockActual: 15, Categoria: "Almacén"},
			{Nombre: "Coca-Cola 1.5L", CodigoBarras: "7790003000104", PrecioCosto: "1200", PrecioVenta: "1800", UnidadMedida: "unidad", StockActual: 20, Categoria: "Bebidas"},
			{Nombre: "Leche Entera 1L", CodigoBarras: "7790003000105", PrecioCosto: "800", PrecioVenta: "1100", UnidadMedida: "unidad", StockActual: 24, Categoria: "Lácteos"},
			{Nombre: "Lavandina 2L", CodigoBarras: "7790003000106", PrecioCosto: "700", PrecioVenta: "1100", UnidadMedida: "unidad", StockActual: 10, Categoria: "Limpieza"},
			{Nombre: "Alfajor Triple", CodigoBarras: "7790003000107", PrecioCosto: "400", PrecioVenta: "650", UnidadMedida: "unidad", StockActual: 30, Categoria: "Golosinas"},
			{Nombre: "Hamburguesas x4", CodigoBarras: "7790003000108", PrecioCosto: "2000", PrecioVenta: "2800", UnidadMedida: "unidad", StockActual: 12, Categoria: "Congelados"},
		},
	},
	"verduleria": {
		Label: "Verdulería",
		Categorias: []presetCategory{
			{Nombre: "Frutas"},
			{Nombre: "Verduras"},
			{Nombre: "Legumbres"},
			{Nombre: "Hierbas"},
		},
		Productos: []presetProduct{
			{Nombre: "Tomate kg", CodigoBarras: "7790004000101", PrecioCosto: "1200", PrecioVenta: "1800", UnidadMedida: "kg", StockActual: 30, Categoria: "Verduras"},
			{Nombre: "Papa kg", CodigoBarras: "7790004000102", PrecioCosto: "600", PrecioVenta: "900", UnidadMedida: "kg", StockActual: 50, Categoria: "Verduras"},
			{Nombre: "Cebolla kg", CodigoBarras: "7790004000103", PrecioCosto: "500", PrecioVenta: "800", UnidadMedida: "kg", StockActual: 40, Categoria: "Verduras"},
			{Nombre: "Banana kg", CodigoBarras: "7790004000104", PrecioCosto: "1000", PrecioVenta: "1500", UnidadMedida: "kg", StockActual: 25, Categoria: "Frutas"},
			{Nombre: "Manzana kg", CodigoBarras: "7790004000105", PrecioCosto: "1200", PrecioVenta: "1800", UnidadMedida: "kg", StockActual: 20, Categoria: "Frutas"},
			{Nombre: "Lechuga unidad", CodigoBarras: "7790004000106", PrecioCosto: "500", PrecioVenta: "800", UnidadMedida: "unidad", StockActual: 15, Categoria: "Verduras"},
			{Nombre: "Limón kg", CodigoBarras: "7790004000107", PrecioCosto: "800", PrecioVenta: "1200", UnidadMedida: "kg", StockActual: 20, Categoria: "Frutas"},
		},
	},
}

// ── Public API ──────────────────────────────────────────────────────────────

// GetPresetInfo returns the preset summary for a given business type.
// Used by the public preview endpoint.
func GetPresetInfo(tipo string) (*dto.PresetResponse, error) {
	p, ok := presets[tipo]
	if !ok {
		return nil, fmt.Errorf("tipo de negocio desconocido: %s", tipo)
	}

	// Count products per category
	catCount := make(map[string]int)
	for _, prod := range p.Productos {
		catCount[prod.Categoria]++
	}

	cats := make([]dto.PresetCategoryResponse, len(p.Categorias))
	for i, c := range p.Categorias {
		cats[i] = dto.PresetCategoryResponse{
			Nombre:       c.Nombre,
			ProductCount: catCount[c.Nombre],
		}
	}

	return &dto.PresetResponse{
		TipoNegocio:     tipo,
		Label:           p.Label,
		TotalCategorias: len(p.Categorias),
		TotalProductos:  len(p.Productos),
		Categorias:      cats,
	}, nil
}

// GetAllPresetSummaries returns a summary for every available business type.
func GetAllPresetSummaries() []dto.PresetResponse {
	order := []string{"kiosco", "carniceria", "minimarket", "verduleria"}
	result := make([]dto.PresetResponse, 0, len(order))
	for _, tipo := range order {
		info, err := GetPresetInfo(tipo)
		if err != nil {
			continue
		}
		result = append(result, *info)
	}
	return result
}

// SeedPresets creates preset categories and sample products for a newly registered tenant.
// It uses the raw *gorm.DB to bypass the tenant-scoped middleware (the tenant was just created
// and there is no JWT context yet). This is best-effort: failures are logged, not propagated.
func SeedPresets(db *gorm.DB, tenantID uuid.UUID, tipoNegocio string) {
	if tipoNegocio == "" {
		tipoNegocio = "kiosco"
	}

	p, ok := presets[tipoNegocio]
	if !ok {
		log.Warn().Str("tipo_negocio", tipoNegocio).Msg("preset not found, skipping seed")
		return
	}

	// Build a context with the tenant ID for logging, though we use raw DB directly.
	ctx := context.WithValue(context.Background(), tenantctx.Key, tenantID)
	_ = ctx // used for future logging enrichment

	// Create categories and collect name → ID mapping
	catMap := make(map[string]uuid.UUID)
	for _, cat := range p.Categorias {
		c := model.Categoria{
			TenantID: tenantID,
			Nombre:   cat.Nombre,
			Activo:   true,
		}
		if err := db.Create(&c).Error; err != nil {
			log.Error().Err(err).
				Str("tenant_id", tenantID.String()).
				Str("categoria", cat.Nombre).
				Msg("failed to seed category")
			continue
		}
		catMap[cat.Nombre] = c.ID
	}

	// Create products
	for _, prod := range p.Productos {
		catID, ok := catMap[prod.Categoria]
		if !ok {
			log.Warn().
				Str("tenant_id", tenantID.String()).
				Str("producto", prod.Nombre).
				Str("categoria", prod.Categoria).
				Msg("category not found for product, skipping")
			continue
		}

		costo := decimal.RequireFromString(prod.PrecioCosto)
		venta := decimal.RequireFromString(prod.PrecioVenta)
		margen := decimal.Zero
		if !costo.IsZero() {
			margen = venta.Sub(costo).Div(costo).Mul(decimal.NewFromInt(100)).Round(2)
		}

		producto := model.Producto{
			TenantID:     tenantID,
			CodigoBarras: prod.CodigoBarras,
			Nombre:       prod.Nombre,
			Categoria:    prod.Categoria,
			CategoriaID:  catID,
			PrecioCosto:  costo,
			PrecioVenta:  venta,
			MargenPct:    margen,
			StockActual:  prod.StockActual,
			StockMinimo:  5,
			UnidadMedida: prod.UnidadMedida,
			Activo:       true,
		}
		if err := db.Create(&producto).Error; err != nil {
			log.Error().Err(err).
				Str("tenant_id", tenantID.String()).
				Str("producto", prod.Nombre).
				Msg("failed to seed product")
		}
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("tipo_negocio", tipoNegocio).
		Int("categorias", len(p.Categorias)).
		Int("productos", len(p.Productos)).
		Msg("preset seed completed")
}
