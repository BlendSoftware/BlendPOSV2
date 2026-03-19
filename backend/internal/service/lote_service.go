package service

import (
	"context"
	"fmt"
	"time"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"

	"github.com/google/uuid"
)

// LoteService defines the contract for product lot/batch management.
type LoteService interface {
	CrearLote(ctx context.Context, req dto.CrearLoteRequest) (*dto.LoteResponse, error)
	ListarLotes(ctx context.Context, productoID uuid.UUID) ([]dto.LoteResponse, error)
	EliminarLote(ctx context.Context, id uuid.UUID) error
	ObtenerAlertasVencimiento(ctx context.Context, dias int) ([]dto.AlertaVencimientoResponse, error)
}

type loteService struct {
	loteRepo    repository.LoteRepository
	productoRepo repository.ProductoRepository
}

func NewLoteService(loteRepo repository.LoteRepository, productoRepo repository.ProductoRepository) LoteService {
	return &loteService{loteRepo: loteRepo, productoRepo: productoRepo}
}

func (s *loteService) CrearLote(ctx context.Context, req dto.CrearLoteRequest) (*dto.LoteResponse, error) {
	productoID, err := uuid.Parse(req.ProductoID)
	if err != nil {
		return nil, fmt.Errorf("producto_id inválido: %w", err)
	}

	// Validate product exists and has controla_vencimiento enabled
	producto, err := s.productoRepo.FindByID(ctx, productoID)
	if err != nil {
		return nil, fmt.Errorf("producto no encontrado: %w", err)
	}
	if !producto.ControlaVencimiento {
		return nil, fmt.Errorf("el producto '%s' no tiene habilitado el control de vencimiento", producto.Nombre)
	}

	fechaVenc, err := time.Parse("2006-01-02", req.FechaVencimiento)
	if err != nil {
		return nil, fmt.Errorf("fecha_vencimiento inválida, usar formato YYYY-MM-DD: %w", err)
	}

	if req.Cantidad < 1 {
		return nil, fmt.Errorf("la cantidad debe ser al menos 1")
	}

	lote := &model.LoteProducto{
		ProductoID:       productoID,
		CodigoLote:       req.CodigoLote,
		FechaVencimiento: fechaVenc,
		Cantidad:         req.Cantidad,
	}

	if err := s.loteRepo.Create(ctx, lote); err != nil {
		return nil, err
	}

	resp := toLoteResponse(lote, producto.Nombre)
	return &resp, nil
}

func (s *loteService) ListarLotes(ctx context.Context, productoID uuid.UUID) ([]dto.LoteResponse, error) {
	lotes, err := s.loteRepo.ListByProducto(ctx, productoID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.LoteResponse, 0, len(lotes))
	for _, l := range lotes {
		nombre := ""
		if l.Producto != nil {
			nombre = l.Producto.Nombre
		}
		result = append(result, toLoteResponse(&l, nombre))
	}
	return result, nil
}

func (s *loteService) EliminarLote(ctx context.Context, id uuid.UUID) error {
	return s.loteRepo.Delete(ctx, id)
}

func (s *loteService) ObtenerAlertasVencimiento(ctx context.Context, dias int) ([]dto.AlertaVencimientoResponse, error) {
	alertas, err := s.loteRepo.GetAlertasVencimiento(ctx, dias)
	if err != nil {
		return nil, err
	}

	result := make([]dto.AlertaVencimientoResponse, 0, len(alertas))
	for _, a := range alertas {
		nombre := ""
		if a.Producto != nil {
			nombre = a.Producto.Nombre
		}
		result = append(result, dto.AlertaVencimientoResponse{
			ID:               a.ID.String(),
			ProductoID:       a.ProductoID.String(),
			ProductoNombre:   nombre,
			CodigoLote:       a.CodigoLote,
			FechaVencimiento: a.FechaVencimiento.Format("2006-01-02"),
			DiasRestantes:    a.DiasRestantes,
			Cantidad:         a.Cantidad,
			Estado:           a.Estado,
		})
	}
	return result, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func toLoteResponse(l *model.LoteProducto, productoNombre string) dto.LoteResponse {
	return dto.LoteResponse{
		ID:               l.ID.String(),
		ProductoID:       l.ProductoID.String(),
		ProductoNombre:   productoNombre,
		CodigoLote:       l.CodigoLote,
		FechaVencimiento: l.FechaVencimiento.Format("2006-01-02"),
		Cantidad:         l.Cantidad,
		CreatedAt:        l.CreatedAt.Format(time.RFC3339),
	}
}
