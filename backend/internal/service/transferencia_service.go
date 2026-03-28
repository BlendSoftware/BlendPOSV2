package service

import (
	"context"
	"errors"
	"fmt"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransferenciaService defines the contract for stock transfer operations.
type TransferenciaService interface {
	CrearTransferencia(ctx context.Context, usuarioID uuid.UUID, req dto.CrearTransferenciaRequest) (*dto.TransferenciaResponse, error)
	CompletarTransferencia(ctx context.Context, id uuid.UUID, usuarioID uuid.UUID) (*dto.TransferenciaResponse, error)
	RechazarTransferencia(ctx context.Context, id uuid.UUID) error
	CancelarTransferencia(ctx context.Context, id uuid.UUID) error
	ListarTransferencias(ctx context.Context, estado string) (*dto.TransferenciaListResponse, error)
	ObtenerTransferencia(ctx context.Context, id uuid.UUID) (*dto.TransferenciaResponse, error)

	// Stock sucursal operations
	ListarStockSucursal(ctx context.Context, sucursalID uuid.UUID) (*dto.StockSucursalListResponse, error)
	AjustarStockSucursal(ctx context.Context, req dto.AjustarStockSucursalRequest) error
	GetAlertasBySucursal(ctx context.Context, sucursalID uuid.UUID) ([]dto.StockSucursalResponse, error)
}

type transferenciaService struct {
	transRepo    repository.TransferenciaRepository
	stockRepo    repository.StockSucursalRepository
	sucursalRepo repository.SucursalRepository
}

func NewTransferenciaService(
	transRepo repository.TransferenciaRepository,
	stockRepo repository.StockSucursalRepository,
	sucursalRepo repository.SucursalRepository,
) TransferenciaService {
	return &transferenciaService{
		transRepo:    transRepo,
		stockRepo:    stockRepo,
		sucursalRepo: sucursalRepo,
	}
}

// ── CrearTransferencia ──────────────────────────────────────────────────────

func (s *transferenciaService) CrearTransferencia(ctx context.Context, usuarioID uuid.UUID, req dto.CrearTransferenciaRequest) (*dto.TransferenciaResponse, error) {
	origenID, err := uuid.Parse(req.SucursalOrigenID)
	if err != nil {
		return nil, fmt.Errorf("sucursal_origen_id inválido: %w", err)
	}
	destinoID, err := uuid.Parse(req.SucursalDestinoID)
	if err != nil {
		return nil, fmt.Errorf("sucursal_destino_id inválido: %w", err)
	}
	if origenID == destinoID {
		return nil, errors.New("sucursal origen y destino no pueden ser la misma")
	}

	// Validate both sucursales exist
	if _, err := s.sucursalRepo.FindByID(ctx, origenID); err != nil {
		return nil, fmt.Errorf("sucursal origen no encontrada: %w", err)
	}
	if _, err := s.sucursalRepo.FindByID(ctx, destinoID); err != nil {
		return nil, fmt.Errorf("sucursal destino no encontrada: %w", err)
	}

	// Validate stock available at origin for each item.
	// Use GetOrCreateStock so that products without an explicit stock_sucursal
	// record get one auto-created (with stock_actual=0) instead of failing.
	for _, item := range req.Items {
		productoID, err := uuid.Parse(item.ProductoID)
		if err != nil {
			return nil, fmt.Errorf("producto_id inválido: %w", err)
		}
		stock, err := s.stockRepo.GetOrCreateStock(ctx, productoID, origenID)
		if err != nil {
			return nil, fmt.Errorf("error verificando stock para producto %s en sucursal origen: %w", item.ProductoID, err)
		}
		if stock.StockActual < item.Cantidad {
			return nil, fmt.Errorf("stock insuficiente en origen para producto %s: disponible %d, solicitado %d",
				item.ProductoID, stock.StockActual, item.Cantidad)
		}
	}

	// Build model
	transfer := &model.TransferenciaStock{
		SucursalOrigenID:  origenID,
		SucursalDestinoID: destinoID,
		Estado:            "pendiente",
		Notas:             req.Notas,
		CreadoPor:         usuarioID,
	}
	for _, item := range req.Items {
		pid, _ := uuid.Parse(item.ProductoID)
		transfer.Items = append(transfer.Items, model.TransferenciaItem{
			ProductoID: pid,
			Cantidad:   item.Cantidad,
		})
	}

	if err := s.transRepo.Create(ctx, transfer); err != nil {
		return nil, err
	}

	// Re-fetch with preloads
	created, err := s.transRepo.FindByID(ctx, transfer.ID)
	if err != nil {
		return nil, err
	}
	return transferenciaToResponse(created), nil
}

// ── CompletarTransferencia ──────────────────────────────────────────────────
// Deduct from origin, add to destination — all in a single transaction.

func (s *transferenciaService) CompletarTransferencia(ctx context.Context, id uuid.UUID, usuarioID uuid.UUID) (*dto.TransferenciaResponse, error) {
	transfer, err := s.transRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("transferencia no encontrada")
	}
	if transfer.Estado != "pendiente" {
		return nil, fmt.Errorf("solo se pueden completar transferencias en estado 'pendiente', estado actual: %s", transfer.Estado)
	}

	db := s.transRepo.DB()
	txErr := runTx(ctx, db, func(tx *gorm.DB) error {
		// For each item: deduct from origen, add to destino
		for _, item := range transfer.Items {
			// Ensure stock rows exist before adjusting
			if _, err := s.stockRepo.GetOrCreateStock(ctx, item.ProductoID, transfer.SucursalOrigenID); err != nil {
				return fmt.Errorf("error asegurando stock origen para producto %s: %w", item.ProductoID.String(), err)
			}
			if _, err := s.stockRepo.GetOrCreateStock(ctx, item.ProductoID, transfer.SucursalDestinoID); err != nil {
				return fmt.Errorf("error asegurando stock destino para producto %s: %w", item.ProductoID.String(), err)
			}

			// Deduct from origin
			if err := s.stockRepo.AjustarStockSucursalTx(tx, item.ProductoID, transfer.SucursalOrigenID, -item.Cantidad); err != nil {
				return fmt.Errorf("error descontando stock de origen: %w", err)
			}
			// Add to destination
			if err := s.stockRepo.AjustarStockSucursalTx(tx, item.ProductoID, transfer.SucursalDestinoID, item.Cantidad); err != nil {
				return fmt.Errorf("error agregando stock a destino: %w", err)
			}
		}

		// Update estado
		return s.transRepo.UpdateEstadoTx(tx, id, "completada", &usuarioID)
	})
	if txErr != nil {
		return nil, txErr
	}

	// Re-fetch with updated state
	updated, err := s.transRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return transferenciaToResponse(updated), nil
}

// ── RechazarTransferencia ───────────────────────────────────────────────────

func (s *transferenciaService) RechazarTransferencia(ctx context.Context, id uuid.UUID) error {
	transfer, err := s.transRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New("transferencia no encontrada")
	}
	if transfer.Estado != "pendiente" {
		return fmt.Errorf("solo se pueden rechazar transferencias en estado 'pendiente', estado actual: %s", transfer.Estado)
	}
	return s.transRepo.UpdateEstado(ctx, id, "rechazada")
}

// ── CancelarTransferencia ───────────────────────────────────────────────────

func (s *transferenciaService) CancelarTransferencia(ctx context.Context, id uuid.UUID) error {
	transfer, err := s.transRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New("transferencia no encontrada")
	}
	if transfer.Estado != "pendiente" {
		return fmt.Errorf("solo se pueden cancelar transferencias en estado 'pendiente', estado actual: %s", transfer.Estado)
	}
	return s.transRepo.UpdateEstado(ctx, id, "cancelada")
}

// ── ListarTransferencias ────────────────────────────────────────────────────

func (s *transferenciaService) ListarTransferencias(ctx context.Context, estado string) (*dto.TransferenciaListResponse, error) {
	items, total, err := s.transRepo.List(ctx, estado)
	if err != nil {
		return nil, err
	}
	data := make([]dto.TransferenciaResponse, 0, len(items))
	for i := range items {
		data = append(data, *transferenciaToResponse(&items[i]))
	}
	return &dto.TransferenciaListResponse{Data: data, Total: total}, nil
}

// ── ObtenerTransferencia ────────────────────────────────────────────────────

func (s *transferenciaService) ObtenerTransferencia(ctx context.Context, id uuid.UUID) (*dto.TransferenciaResponse, error) {
	t, err := s.transRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("transferencia no encontrada")
	}
	return transferenciaToResponse(t), nil
}

// ── Stock Sucursal operations ───────────────────────────────────────────────

func (s *transferenciaService) ListarStockSucursal(ctx context.Context, sucursalID uuid.UUID) (*dto.StockSucursalListResponse, error) {
	items, total, err := s.stockRepo.ListBySucursal(ctx, sucursalID)
	if err != nil {
		return nil, err
	}
	data := make([]dto.StockSucursalResponse, 0, len(items))
	for _, item := range items {
		data = append(data, stockSucursalToResponse(&item))
	}
	return &dto.StockSucursalListResponse{Data: data, Total: total}, nil
}

func (s *transferenciaService) AjustarStockSucursal(ctx context.Context, req dto.AjustarStockSucursalRequest) error {
	productoID, err := uuid.Parse(req.ProductoID)
	if err != nil {
		return fmt.Errorf("producto_id inválido: %w", err)
	}
	sucursalID, err := uuid.Parse(req.SucursalID)
	if err != nil {
		return fmt.Errorf("sucursal_id inválido: %w", err)
	}

	// Ensure the stock row exists
	if _, err := s.stockRepo.GetOrCreateStock(ctx, productoID, sucursalID); err != nil {
		return fmt.Errorf("error asegurando registro de stock: %w", err)
	}

	return s.stockRepo.AjustarStockSucursal(ctx, productoID, sucursalID, req.Delta)
}

func (s *transferenciaService) GetAlertasBySucursal(ctx context.Context, sucursalID uuid.UUID) ([]dto.StockSucursalResponse, error) {
	items, err := s.stockRepo.GetAlertasBySucursal(ctx, sucursalID)
	if err != nil {
		return nil, err
	}
	data := make([]dto.StockSucursalResponse, 0, len(items))
	for _, item := range items {
		data = append(data, stockSucursalToResponse(&item))
	}
	return data, nil
}

// ── Mappers ─────────────────────────────────────────────────────────────────

func transferenciaToResponse(t *model.TransferenciaStock) *dto.TransferenciaResponse {
	resp := &dto.TransferenciaResponse{
		ID:                t.ID.String(),
		SucursalOrigenID:  t.SucursalOrigenID.String(),
		SucursalDestinoID: t.SucursalDestinoID.String(),
		Estado:            t.Estado,
		Notas:             t.Notas,
		CreadoPor:         t.CreadoPor.String(),
		CreatedAt:         t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if t.SucursalOrigen != nil {
		resp.SucursalOrigen = t.SucursalOrigen.Nombre
	}
	if t.SucursalDestino != nil {
		resp.SucursalDestino = t.SucursalDestino.Nombre
	}
	if t.Creador != nil {
		resp.CreadoPorNombre = t.Creador.Nombre
	}
	if t.CompletadoPor != nil {
		s := t.CompletadoPor.String()
		resp.CompletadoPor = &s
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.Format("2006-01-02T15:04:05Z")
		resp.CompletedAt = &s
	}

	resp.Items = make([]dto.TransferenciaItemResponse, 0, len(t.Items))
	for _, item := range t.Items {
		nombre := ""
		if item.Producto != nil {
			nombre = item.Producto.Nombre
		}
		resp.Items = append(resp.Items, dto.TransferenciaItemResponse{
			ID:         item.ID.String(),
			ProductoID: item.ProductoID.String(),
			Producto:   nombre,
			Cantidad:   item.Cantidad,
		})
	}
	return resp
}

func stockSucursalToResponse(ss *model.StockSucursal) dto.StockSucursalResponse {
	nombre := ""
	if ss.Producto != nil {
		nombre = ss.Producto.Nombre
	}
	return dto.StockSucursalResponse{
		ID:          ss.ID.String(),
		ProductoID:  ss.ProductoID.String(),
		Producto:    nombre,
		SucursalID:  ss.SucursalID.String(),
		StockActual: ss.StockActual,
		StockMinimo: ss.StockMinimo,
		UpdatedAt:   ss.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
