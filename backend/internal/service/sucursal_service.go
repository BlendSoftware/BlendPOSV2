package service

import (
	"context"
	"errors"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"

	"github.com/google/uuid"
)

type SucursalService interface {
	Crear(ctx context.Context, req dto.CrearSucursalRequest) (*dto.SucursalResponse, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.SucursalResponse, error)
	Actualizar(ctx context.Context, id uuid.UUID, req dto.UpdateSucursalRequest) (*dto.SucursalResponse, error)
	Listar(ctx context.Context, incluirInactivas bool) (*dto.SucursalListResponse, error)
}

type sucursalService struct {
	repo repository.SucursalRepository
}

func NewSucursalService(repo repository.SucursalRepository) SucursalService {
	return &sucursalService{repo: repo}
}

func (s *sucursalService) Crear(ctx context.Context, req dto.CrearSucursalRequest) (*dto.SucursalResponse, error) {
	suc := &model.Sucursal{
		Nombre:    req.Nombre,
		Direccion: req.Direccion,
		Telefono:  req.Telefono,
		Activa:    true,
	}
	if err := s.repo.Create(ctx, suc); err != nil {
		return nil, err
	}
	return sucursalToResponse(suc), nil
}

func (s *sucursalService) ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.SucursalResponse, error) {
	suc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("sucursal no encontrada")
	}
	return sucursalToResponse(suc), nil
}

func (s *sucursalService) Actualizar(ctx context.Context, id uuid.UUID, req dto.UpdateSucursalRequest) (*dto.SucursalResponse, error) {
	suc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("sucursal no encontrada")
	}
	if req.Nombre != nil {
		suc.Nombre = *req.Nombre
	}
	if req.Direccion != nil {
		suc.Direccion = req.Direccion
	}
	if req.Telefono != nil {
		suc.Telefono = req.Telefono
	}
	if req.Activa != nil {
		suc.Activa = *req.Activa
	}
	if req.EsDeposito != nil {
		suc.EsDeposito = *req.EsDeposito
	}
	if err := s.repo.Update(ctx, suc); err != nil {
		return nil, err
	}
	return sucursalToResponse(suc), nil
}

func (s *sucursalService) Listar(ctx context.Context, incluirInactivas bool) (*dto.SucursalListResponse, error) {
	sucursales, total, err := s.repo.List(ctx, incluirInactivas)
	if err != nil {
		return nil, err
	}
	data := make([]dto.SucursalResponse, 0, len(sucursales))
	for _, suc := range sucursales {
		data = append(data, *sucursalToResponse(&suc))
	}
	return &dto.SucursalListResponse{Data: data, Total: total}, nil
}

// ── Mapper ──────────────────────────────────────────────────────────────────

func sucursalToResponse(s *model.Sucursal) *dto.SucursalResponse {
	return &dto.SucursalResponse{
		ID:         s.ID.String(),
		Nombre:     s.Nombre,
		Direccion:  s.Direccion,
		Telefono:   s.Telefono,
		Activa:     s.Activa,
		EsDeposito: s.EsDeposito,
		CreatedAt:  s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
