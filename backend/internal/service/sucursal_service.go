package service

import (
	"context"
	"errors"
	"fmt"

	"blendpos/internal/dto"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/tenantctx"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type SucursalService interface {
	Crear(ctx context.Context, req dto.CrearSucursalRequest) (*dto.SucursalResponse, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*dto.SucursalResponse, error)
	Actualizar(ctx context.Context, id uuid.UUID, req dto.UpdateSucursalRequest) (*dto.SucursalResponse, error)
	Listar(ctx context.Context, incluirInactivas bool) (*dto.SucursalListResponse, error)
	// EnsureDefaultSucursal creates a "Casa Central" sucursal for a tenant if none exists.
	// Used during tenant registration. Operates with raw *gorm.DB (no JWT context).
	EnsureDefaultSucursal(db *gorm.DB, tenantID uuid.UUID) error
}

type sucursalService struct {
	repo     repository.SucursalRepository
	planRepo repository.TenantRepository
}

func NewSucursalService(repo repository.SucursalRepository, planRepo ...repository.TenantRepository) SucursalService {
	svc := &sucursalService{repo: repo}
	if len(planRepo) > 0 {
		svc.planRepo = planRepo[0]
	}
	return svc
}

func (s *sucursalService) Crear(ctx context.Context, req dto.CrearSucursalRequest) (*dto.SucursalResponse, error) {
	// Enforce plan limit on max_sucursales
	if s.planRepo != nil {
		if err := s.enforcePlanLimit(ctx); err != nil {
			return nil, err
		}
	}

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

// enforcePlanLimit checks whether the tenant has reached their plan's max_sucursales limit.
func (s *sucursalService) enforcePlanLimit(ctx context.Context) error {
	count, err := s.repo.CountActiveByTenant(ctx)
	if err != nil {
		return fmt.Errorf("error contando sucursales: %w", err)
	}

	// Get tenant plan limits
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	tenant, err := s.planRepo.FindTenantByID(ctx, tid)
	if err != nil {
		return fmt.Errorf("error obteniendo tenant: %w", err)
	}
	if tenant.Plan == nil {
		// No plan assigned — allow creation
		return nil
	}

	maxSucursales := tenant.Plan.MaxSucursales
	if maxSucursales > 0 && count >= int64(maxSucursales) {
		return fmt.Errorf("tu plan %s permite máximo %d sucursal(es). Actualizá tu plan para crear más", tenant.Plan.Nombre, maxSucursales)
	}

	return nil
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

// EnsureDefaultSucursal creates a "Casa Central" sucursal for a newly registered tenant.
// Operates outside of JWT context (raw DB) because this runs during registration.
func (s *sucursalService) EnsureDefaultSucursal(db *gorm.DB, tenantID uuid.UUID) error {
	// Check if any sucursal already exists for this tenant
	var count int64
	if err := db.Model(&model.Sucursal{}).Where("tenant_id = ?", tenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("error checking existing sucursales: %w", err)
	}
	if count > 0 {
		return nil // Already has a sucursal
	}

	suc := &model.Sucursal{
		TenantID: tenantID,
		Nombre:   "Casa Central",
		Activa:   true,
	}
	if err := s.repo.CreateWithDB(db, suc); err != nil {
		return fmt.Errorf("error creating default sucursal: %w", err)
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("sucursal_id", suc.ID.String()).
		Msg("default sucursal 'Casa Central' created for new tenant")

	return nil
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
