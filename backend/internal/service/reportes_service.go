package service

import (
	"context"
	"errors"
	"time"

	"blendpos/internal/dto"
	"blendpos/internal/repository"

	"github.com/google/uuid"
)

// ReportesService provides analytics/reporting business logic.
// All methods accept an optional sucursalID — nil means consolidated (all branches).
type ReportesService interface {
	GetVentasResumen(ctx context.Context, tenantID uuid.UUID, desde, hasta string, sucursalID *uuid.UUID) (*dto.VentasResumenResponse, error)
	GetTopProductos(ctx context.Context, tenantID uuid.UUID, desde, hasta string, limit int, sucursalID *uuid.UUID) ([]dto.TopProductoResponse, error)
	GetVentasPorMedioPago(ctx context.Context, tenantID uuid.UUID, desde, hasta string, sucursalID *uuid.UUID) ([]dto.VentasPorMedioPagoResponse, error)
	GetVentasPorPeriodo(ctx context.Context, tenantID uuid.UUID, desde, hasta, agrupacion string, sucursalID *uuid.UUID) ([]dto.VentasPorPeriodoResponse, error)
	GetVentasPorCajero(ctx context.Context, tenantID uuid.UUID, desde, hasta string, sucursalID *uuid.UUID) ([]dto.ReporteCajeroResponse, error)
	GetReporteTurnos(ctx context.Context, tenantID uuid.UUID, desde, hasta string, sucursalID *uuid.UUID) ([]dto.ReporteTurnoResponse, error)
}

type reportesService struct {
	repo repository.IReportesRepo
}

func NewReportesService(repo repository.IReportesRepo) ReportesService {
	return &reportesService{repo: repo}
}

const maxRangeDays = 366 // max 1 year range

// parseDateRange validates and parses desde/hasta strings into time.Time values.
// Returns half-open interval [desde, hastaExclusive) where hastaExclusive = hasta + 1 day.
func parseDateRange(desdeStr, hastaStr string) (time.Time, time.Time, error) {
	desde, err := time.Parse("2006-01-02", desdeStr)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("parámetro 'desde' inválido, use YYYY-MM-DD")
	}
	hasta, err := time.Parse("2006-01-02", hastaStr)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("parámetro 'hasta' inválido, use YYYY-MM-DD")
	}
	if desde.After(hasta) {
		return time.Time{}, time.Time{}, errors.New("'desde' no puede ser posterior a 'hasta'")
	}
	if hasta.Sub(desde).Hours()/24 > float64(maxRangeDays) {
		return time.Time{}, time.Time{}, errors.New("rango máximo permitido: 1 año (366 días)")
	}
	// Exclusive upper bound: start of the day after 'hasta'
	hastaExclusive := hasta.AddDate(0, 0, 1)
	return desde, hastaExclusive, nil
}

func (s *reportesService) GetVentasResumen(ctx context.Context, tenantID uuid.UUID, desdeStr, hastaStr string, sucursalID *uuid.UUID) (*dto.VentasResumenResponse, error) {
	desde, hasta, err := parseDateRange(desdeStr, hastaStr)
	if err != nil {
		return nil, err
	}
	res, err := s.repo.GetVentasResumen(ctx, tenantID, desde, hasta, sucursalID)
	if err != nil {
		return nil, err
	}
	res.PeriodoDesde = desdeStr
	res.PeriodoHasta = hastaStr
	return res, nil
}

func (s *reportesService) GetTopProductos(ctx context.Context, tenantID uuid.UUID, desdeStr, hastaStr string, limit int, sucursalID *uuid.UUID) ([]dto.TopProductoResponse, error) {
	desde, hasta, err := parseDateRange(desdeStr, hastaStr)
	if err != nil {
		return nil, err
	}
	return s.repo.GetTopProductos(ctx, tenantID, desde, hasta, limit, sucursalID)
}

func (s *reportesService) GetVentasPorMedioPago(ctx context.Context, tenantID uuid.UUID, desdeStr, hastaStr string, sucursalID *uuid.UUID) ([]dto.VentasPorMedioPagoResponse, error) {
	desde, hasta, err := parseDateRange(desdeStr, hastaStr)
	if err != nil {
		return nil, err
	}
	return s.repo.GetVentasPorMedioPago(ctx, tenantID, desde, hasta, sucursalID)
}

func (s *reportesService) GetVentasPorPeriodo(ctx context.Context, tenantID uuid.UUID, desdeStr, hastaStr, agrupacion string, sucursalID *uuid.UUID) ([]dto.VentasPorPeriodoResponse, error) {
	desde, hasta, err := parseDateRange(desdeStr, hastaStr)
	if err != nil {
		return nil, err
	}
	// Validate agrupacion
	switch agrupacion {
	case "dia", "day", "semana", "week", "mes", "month":
		// OK
	default:
		return nil, errors.New("agrupacion debe ser 'dia', 'semana' o 'mes'")
	}
	return s.repo.GetVentasPorPeriodo(ctx, tenantID, desde, hasta, agrupacion, sucursalID)
}

func (s *reportesService) GetVentasPorCajero(ctx context.Context, tenantID uuid.UUID, desdeStr, hastaStr string, sucursalID *uuid.UUID) ([]dto.ReporteCajeroResponse, error) {
	desde, hasta, err := parseDateRange(desdeStr, hastaStr)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.GetVentasPorCajero(ctx, tenantID, desde, hasta, sucursalID)
	if err != nil {
		return nil, err
	}
	// Stamp period on each row
	for i := range rows {
		rows[i].PeriodoDesde = desdeStr
		rows[i].PeriodoHasta = hastaStr
	}
	return rows, nil
}

func (s *reportesService) GetReporteTurnos(ctx context.Context, tenantID uuid.UUID, desdeStr, hastaStr string, sucursalID *uuid.UUID) ([]dto.ReporteTurnoResponse, error) {
	desde, hasta, err := parseDateRange(desdeStr, hastaStr)
	if err != nil {
		return nil, err
	}
	return s.repo.GetReporteTurnos(ctx, tenantID, desde, hasta, sucursalID)
}
