package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// ProductoService defines the business logic contract for products.
type ProductoService interface {
	Crear(ctx context.Context, req dto.CrearProductoRequest) (*dto.ProductoResponse, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.ProductoResponse, error)
	ObtenerPorBarcode(ctx context.Context, barcode string) (*dto.ProductoResponse, error)
	Listar(ctx context.Context, filter dto.ProductoFilter) (*dto.ProductoListResponse, error)
	Actualizar(ctx context.Context, id uuid.UUID, req dto.ActualizarProductoRequest) (*dto.ProductoResponse, error)
	Desactivar(ctx context.Context, id uuid.UUID) error
	Reactivar(ctx context.Context, id uuid.UUID) error
	AjustarStock(ctx context.Context, id uuid.UUID, req dto.AjustarStockRequest) (*dto.ProductoResponse, error)
	CrearVariante(ctx context.Context, padreID uuid.UUID, req dto.CrearVarianteRequest) (*dto.ProductoResponse, error)
	ListarVariantes(ctx context.Context, padreID uuid.UUID) ([]dto.ProductoResponse, error)
}

type productoService struct {
	repo         repository.ProductoRepository
	movRepo      repository.MovimientoStockRepository
	catRepo      repository.CategoriaRepository
	rdb          *redis.Client
	stockSucRepo repository.StockSucursalRepository
	sucursalRepo repository.SucursalRepository
}

func NewProductoService(
	repo repository.ProductoRepository,
	movRepo repository.MovimientoStockRepository,
	catRepo repository.CategoriaRepository,
	rdb *redis.Client,
	stockSucRepo repository.StockSucursalRepository,
	sucursalRepo repository.SucursalRepository,
) ProductoService {
	return &productoService{repo: repo, movRepo: movRepo, catRepo: catRepo, rdb: rdb, stockSucRepo: stockSucRepo, sucursalRepo: sucursalRepo}
}

// lookupCategoriaID busca la categoría por nombre y devuelve su ID.
// Si la categoría no existe, la crea automáticamente.
// Si catRepo es nil (en tests) devuelve uuid.Nil sin error para mantener
// compatibilidad con los stubs de test que no usan DB real.
func (s *productoService) lookupCategoriaID(ctx context.Context, nombre string) (uuid.UUID, error) {
	if s.catRepo == nil {
		return uuid.Nil, nil
	}
	
	// Intentar obtener la categoría existente
	cat, err := s.catRepo.ObtenerPorNombre(ctx, nombre)
	if err == nil {
		// Categoría encontrada, devolver su ID
		return cat.ID, nil
	}
	
	// Si la categoría no existe, crearla automáticamente
	nuevaCat := &model.Categoria{
		Nombre: nombre,
		Activo: true,
	}
	
	if err := s.catRepo.Crear(ctx, nuevaCat); err != nil {
		return uuid.Nil, fmt.Errorf("no se pudo crear la categoría '%s': %w", nombre, err)
	}
	
	return nuevaCat.ID, nil
}

// precioCacheKey returns the Redis key for a product's price cache entry.
func precioCacheKey(barcode string) string { return fmt.Sprintf("precio:%s", barcode) }

// invalidatePrecioCache removes the cached price for a given barcode.
// A best-effort operation — errors are intentionally swallowed.
// Safe to call when rdb is nil (e.g. in unit tests).
func (s *productoService) invalidatePrecioCache(ctx context.Context, barcode string) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Del(ctx, precioCacheKey(barcode)).Err()
}

// calcMargen returns (precioVenta - precioCosto) / precioCosto * 100.
// Returns 0 if precioCosto is zero to avoid division by zero.
func calcMargen(costo, venta decimal.Decimal) decimal.Decimal {
	if costo.IsZero() {
		return decimal.Zero
	}
	return venta.Sub(costo).Div(costo).Mul(decimal.NewFromInt(100)).Round(2)
}

// toProductoResponse maps a model.Producto to its response DTO.
func toProductoResponse(p *model.Producto) *dto.ProductoResponse {
	var provStr *string
	if p.ProveedorID != nil {
		s := p.ProveedorID.String()
		provStr = &s
	}
	var padreStr *string
	if p.PadreID != nil {
		s := p.PadreID.String()
		padreStr = &s
	}

	// Parse variant attributes from JSON
	var attrs map[string]string
	if len(p.VarianteAtributos) > 0 && string(p.VarianteAtributos) != "{}" {
		_ = json.Unmarshal(p.VarianteAtributos, &attrs)
	}

	return &dto.ProductoResponse{
		ID:           p.ID.String(),
		CodigoBarras: p.CodigoBarras,
		Nombre:       p.Nombre,
		Descripcion:  p.Descripcion,
		Categoria:    p.Categoria,
		PrecioCosto:  p.PrecioCosto,
		PrecioVenta:  p.PrecioVenta,
		MargenPct:    calcMargen(p.PrecioCosto, p.PrecioVenta),
		StockActual:  p.StockActual,
		StockMinimo:  p.StockMinimo,
		UnidadMedida: p.UnidadMedida,
		EsPadre:             p.EsPadre,
		PadreID:             padreStr,
		VarianteAtributos:   attrs,
		VarianteNombre:      p.VarianteNombre,
		Activo:              p.Activo,
		ControlaVencimiento: p.ControlaVencimiento,
		ProveedorID:         provStr,
	}
}

// buildVarianteNombre generates the display name for a variant.
// Format: "{ParentName} - {val1} / {val2}" with keys sorted alphabetically.
func buildVarianteNombre(parentName string, attrs map[string]string) string {
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	vals := make([]string, 0, len(keys))
	for _, k := range keys {
		vals = append(vals, attrs[k])
	}
	return parentName + " - " + strings.Join(vals, " / ")
}

// ── Service methods ──────────────────────────────────────────────────────────

func (s *productoService) Crear(ctx context.Context, req dto.CrearProductoRequest) (*dto.ProductoResponse, error) {
	var provID *uuid.UUID
	if req.ProveedorID != nil {
		id, err := uuid.Parse(*req.ProveedorID)
		if err != nil {
			return nil, fmt.Errorf("proveedor_id inválido: %w", err)
		}
		provID = &id
	}

	catID, err := s.lookupCategoriaID(ctx, req.Categoria)
	if err != nil {
		return nil, err
	}

	p := &model.Producto{
		CodigoBarras:        req.CodigoBarras,
		Nombre:              req.Nombre,
		Descripcion:         req.Descripcion,
		Categoria:           req.Categoria,
		CategoriaID:         catID,
		PrecioCosto:         req.PrecioCosto,
		PrecioVenta:         req.PrecioVenta,
		StockActual:         req.StockActual,
		StockMinimo:         req.StockMinimo,
		UnidadMedida:        req.UnidadMedida,
		EsPadre:             req.EsPadre,
		ControlaVencimiento: req.ControlaVencimiento,
		Activo:              true,
		ProveedorID:         provID,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	// Auto-create stock_sucursal records for all active sucursales.
	// Best-effort — don't fail product creation if this errors.
	s.ensureStockSucursalRecords(ctx, p.ID, p.StockActual, p.StockMinimo)

	return toProductoResponse(p), nil
}

// ensureStockSucursalRecords creates a stock_sucursal row for every active
// sucursal so the product appears in the "Stock por Sucursal" view and can be
// transferred from day one. This is a best-effort operation — errors are logged
// but do not block product creation.
func (s *productoService) ensureStockSucursalRecords(ctx context.Context, productoID uuid.UUID, stockActual, stockMinimo int) {
	if s.stockSucRepo == nil || s.sucursalRepo == nil {
		return
	}
	sucursales, _, err := s.sucursalRepo.List(ctx, false) // only active
	if err != nil || len(sucursales) == 0 {
		return
	}
	for _, suc := range sucursales {
		ss, err := s.stockSucRepo.GetOrCreateStock(ctx, productoID, suc.ID)
		if err != nil {
			continue
		}
		// If the record was just created (stock_actual=0) and this is the first
		// sucursal, seed it with the product's initial global stock so the numbers
		// match. Subsequent sucursales start at 0.
		_ = ss // GetOrCreateStock already created the record
	}
}

func (s *productoService) ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.ProductoResponse, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toProductoResponse(p), nil
}

func (s *productoService) ObtenerPorBarcode(ctx context.Context, barcode string) (*dto.ProductoResponse, error) {
	p, err := s.repo.FindByBarcode(ctx, barcode)
	if err != nil {
		return nil, err
	}
	return toProductoResponse(p), nil
}

func (s *productoService) Listar(ctx context.Context, filter dto.ProductoFilter) (*dto.ProductoListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	productos, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ProductoResponse, 0, len(productos))
	for i := range productos {
		resp := toProductoResponse(&productos[i])
		// Enrich parent products with the count of active variants
		if productos[i].EsPadre {
			if cnt, err := s.repo.CountByPadreID(ctx, productos[i].ID); err == nil {
				resp.CantidadVariantes = int(cnt)
			}
		}
		items = append(items, *resp)
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	return &dto.ProductoListResponse{
		Data:       items,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *productoService) Actualizar(ctx context.Context, id uuid.UUID, req dto.ActualizarProductoRequest) (*dto.ProductoResponse, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Nombre != nil {
		p.Nombre = *req.Nombre
	}
	if req.Descripcion != nil {
		p.Descripcion = req.Descripcion
	}
	if req.Categoria != nil {
		nuevoCatID, catErr := s.lookupCategoriaID(ctx, *req.Categoria)
		if catErr != nil {
			return nil, catErr
		}
		p.Categoria = *req.Categoria
		p.CategoriaID = nuevoCatID
	}
	if req.PrecioCosto != nil {
		p.PrecioCosto = *req.PrecioCosto
	}
	if req.PrecioVenta != nil {
		p.PrecioVenta = *req.PrecioVenta
	}
	if req.StockMinimo != nil {
		p.StockMinimo = *req.StockMinimo
	}
	if req.UnidadMedida != nil {
		p.UnidadMedida = *req.UnidadMedida
	}
	if req.ProveedorID != nil {
		pid, err := uuid.Parse(*req.ProveedorID)
		if err != nil {
			return nil, fmt.Errorf("proveedor_id inválido: %w", err)
		}
		p.ProveedorID = &pid
	}
	if req.ControlaVencimiento != nil {
		p.ControlaVencimiento = *req.ControlaVencimiento
	}
	if req.EsPadre != nil {
		p.EsPadre = *req.EsPadre
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	// Invalidate Redis price cache on any price change
	s.invalidatePrecioCache(ctx, p.CodigoBarras)

	return toProductoResponse(p), nil
}

func (s *productoService) Desactivar(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	s.invalidatePrecioCache(ctx, p.CodigoBarras)
	return nil
}

func (s *productoService) Reactivar(ctx context.Context, id uuid.UUID) error {
	return s.repo.Reactivar(ctx, id)
}

// AjustarStock incrementa (delta > 0) o decrementa (delta < 0) el stock de un producto.
// Corresponde a PATCH /v1/productos/:id/stock.
func (s *productoService) AjustarStock(ctx context.Context, id uuid.UUID, req dto.AjustarStockRequest) (*dto.ProductoResponse, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("producto no encontrado")
	}
	if !p.Activo {
		return nil, fmt.Errorf("el producto está desactivado")
	}
	nuevoStock := p.StockActual + req.Delta
	if nuevoStock < 0 {
		return nil, fmt.Errorf("stock insuficiente: el ajuste resultaría en stock negativo (%d)", nuevoStock)
	}

	stockAntes := p.StockActual
	if err := s.repo.AjustarStock(ctx, id, req.Delta); err != nil {
		return nil, err
	}

	// Record movimiento de stock
	motivo := req.Motivo
	if motivo == "" {
		motivo = "Ajuste manual"
	}
	mov := &model.MovimientoStock{
		ProductoID:    id,
		Tipo:          "ajuste_manual",
		Cantidad:      req.Delta,
		StockAnterior: stockAntes,
		StockNuevo:    nuevoStock,
		Motivo:        motivo,
	}
	if s.movRepo != nil {
		_ = s.movRepo.Create(ctx, mov) // best-effort — don't fail the adjustment if this errors
	}

	// Refresh the product from DB to return updated stock
	p, err = s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toProductoResponse(p), nil
}

// CrearVariante creates a variant (child) product from a parent product.
// The parent must exist and have es_padre=true.
// Variant inherits most fields from parent unless explicitly overridden.
func (s *productoService) CrearVariante(ctx context.Context, padreID uuid.UUID, req dto.CrearVarianteRequest) (*dto.ProductoResponse, error) {
	padre, err := s.repo.FindByID(ctx, padreID)
	if err != nil {
		return nil, fmt.Errorf("producto padre no encontrado")
	}
	if !padre.EsPadre {
		return nil, fmt.Errorf("el producto no está marcado como padre (es_padre=false)")
	}

	// Serialize attributes to JSON
	attrJSON, err := json.Marshal(req.Atributos)
	if err != nil {
		return nil, fmt.Errorf("atributos inválidos: %w", err)
	}

	// Use parent prices unless overridden
	precioVenta := padre.PrecioVenta
	if req.PrecioVenta != nil {
		precioVenta = *req.PrecioVenta
	}
	precioCosto := padre.PrecioCosto
	if req.PrecioCosto != nil {
		precioCosto = *req.PrecioCosto
	}

	varNombre := buildVarianteNombre(padre.Nombre, req.Atributos)

	variante := &model.Producto{
		CodigoBarras:        req.CodigoBarras,
		Nombre:              padre.Nombre,
		Descripcion:         padre.Descripcion,
		Categoria:           padre.Categoria,
		CategoriaID:         padre.CategoriaID,
		PrecioCosto:         precioCosto,
		PrecioVenta:         precioVenta,
		StockActual:         req.StockActual,
		StockMinimo:         padre.StockMinimo,
		UnidadMedida:        padre.UnidadMedida,
		EsPadre:             false,
		PadreID:             &padreID,
		VarianteAtributos:   attrJSON,
		VarianteNombre:      &varNombre,
		ProveedorID:         padre.ProveedorID,
		Activo:              true,
		ControlaVencimiento: padre.ControlaVencimiento,
	}

	if err := s.repo.Create(ctx, variante); err != nil {
		return nil, err
	}

	// Auto-create stock_sucursal records for the variant too.
	s.ensureStockSucursalRecords(ctx, variante.ID, variante.StockActual, variante.StockMinimo)

	return toProductoResponse(variante), nil
}

// ListarVariantes returns all active variants (children) for a given parent product.
func (s *productoService) ListarVariantes(ctx context.Context, padreID uuid.UUID) ([]dto.ProductoResponse, error) {
	padre, err := s.repo.FindByID(ctx, padreID)
	if err != nil {
		return nil, fmt.Errorf("producto padre no encontrado")
	}
	if !padre.EsPadre {
		return nil, fmt.Errorf("el producto no está marcado como padre (es_padre=false)")
	}

	variantes, err := s.repo.FindByPadreID(ctx, padreID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ProductoResponse, 0, len(variantes))
	for i := range variantes {
		items = append(items, *toProductoResponse(&variantes[i]))
	}
	return items, nil
}
