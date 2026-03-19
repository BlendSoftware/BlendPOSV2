package repository

// reportes_repo.go
// Analytics repository — reads from the read replica when available.
// Uses explicit WHERE tenant_id = ? because the replica does NOT go through
// TenantMiddleware's set_config (RLS is not active on replica connections).

import (
	"context"
	"fmt"
	"time"

	"blendpos/internal/dto"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// IReportesRepo defines read-only analytics queries against ventas data.
// All methods accept an optional sucursalID filter — when non-nil, results are
// scoped to a single branch; when nil, results are consolidated (all branches).
type IReportesRepo interface {
	GetVentasResumen(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) (*dto.VentasResumenResponse, error)
	GetTopProductos(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, limit int, sucursalID *uuid.UUID) ([]dto.TopProductoResponse, error)
	GetVentasPorMedioPago(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) ([]dto.VentasPorMedioPagoResponse, error)
	GetVentasPorPeriodo(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, agrupacion string, sucursalID *uuid.UUID) ([]dto.VentasPorPeriodoResponse, error)
	GetVentasPorCajero(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) ([]dto.ReporteCajeroResponse, error)
	GetReporteTurnos(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) ([]dto.ReporteTurnoResponse, error)
}

type reportesRepo struct {
	db *gorm.DB
}

// NewReportesRepository creates an analytics repo. Pass DBRead (read replica)
// when available; the caller (router.go) falls back to primary DB if nil.
func NewReportesRepository(db *gorm.DB) IReportesRepo {
	return &reportesRepo{db: db}
}

// GetVentasResumen returns SUM(total), COUNT(*), AVG(total) for completed sales.
func (r *reportesRepo) GetVentasResumen(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) (*dto.VentasResumenResponse, error) {
	type result struct {
		TotalVentas    decimal.Decimal
		CantidadVentas int64
	}
	var res result

	q := r.db.WithContext(ctx).
		Table("ventas").
		Select("COALESCE(SUM(total), 0) AS total_ventas, COUNT(*) AS cantidad_ventas").
		Where("tenant_id = ? AND estado = 'completada' AND created_at >= ? AND created_at < ?",
			tenantID, desde, hasta)
	if sucursalID != nil {
		q = q.Where("sucursal_id = ?", *sucursalID)
	}
	err := q.Scan(&res).Error
	if err != nil {
		return nil, fmt.Errorf("GetVentasResumen: %w", err)
	}

	ticketPromedio := decimal.Zero
	if res.CantidadVentas > 0 {
		ticketPromedio = res.TotalVentas.Div(decimal.NewFromInt(res.CantidadVentas)).RoundBank(2)
	}

	return &dto.VentasResumenResponse{
		TotalVentas:    res.TotalVentas,
		CantidadVentas: res.CantidadVentas,
		TicketPromedio: ticketPromedio,
	}, nil
}

// GetTopProductos returns top-selling products by quantity, joining venta_items + productos.
func (r *reportesRepo) GetTopProductos(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, limit int, sucursalID *uuid.UUID) ([]dto.TopProductoResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	type row struct {
		ProductoID      uuid.UUID
		Nombre          string
		CantidadVendida int64
		TotalRecaudado  decimal.Decimal
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Table("venta_items vi").
		Select("vi.producto_id, p.nombre, SUM(vi.cantidad) AS cantidad_vendida, SUM(vi.subtotal) AS total_recaudado").
		Joins("JOIN ventas v ON v.id = vi.venta_id AND v.tenant_id = ?", tenantID).
		Joins("JOIN productos p ON p.id = vi.producto_id AND p.tenant_id = ?", tenantID).
		Where("vi.tenant_id = ? AND v.estado = 'completada' AND v.created_at >= ? AND v.created_at < ?",
			tenantID, desde, hasta)
	if sucursalID != nil {
		q = q.Where("v.sucursal_id = ?", *sucursalID)
	}
	err := q.Group("vi.producto_id, p.nombre").
		Order("cantidad_vendida DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("GetTopProductos: %w", err)
	}

	result := make([]dto.TopProductoResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.TopProductoResponse{
			ProductoID:      row.ProductoID.String(),
			Nombre:          row.Nombre,
			CantidadVendida: row.CantidadVendida,
			TotalRecaudado:  row.TotalRecaudado,
		})
	}
	return result, nil
}

// GetVentasPorMedioPago groups completed sales by payment method (venta_pagos).
func (r *reportesRepo) GetVentasPorMedioPago(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) ([]dto.VentasPorMedioPagoResponse, error) {
	type row struct {
		MedioPago string
		Cantidad  int64
		Total     decimal.Decimal
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Table("venta_pagos vp").
		Select("vp.metodo AS medio_pago, COUNT(*) AS cantidad, COALESCE(SUM(vp.monto), 0) AS total").
		Joins("JOIN ventas v ON v.id = vp.venta_id AND v.tenant_id = ?", tenantID).
		Where("v.estado = 'completada' AND v.created_at >= ? AND v.created_at < ?",
			desde, hasta)
	if sucursalID != nil {
		q = q.Where("v.sucursal_id = ?", *sucursalID)
	}
	err := q.Group("vp.metodo").
		Order("total DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("GetVentasPorMedioPago: %w", err)
	}

	result := make([]dto.VentasPorMedioPagoResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.VentasPorMedioPagoResponse{
			MedioPago: row.MedioPago,
			Cantidad:  row.Cantidad,
			Total:     row.Total,
		})
	}
	return result, nil
}

// GetVentasPorPeriodo groups sales by date_trunc bucket (dia/semana/mes).
func (r *reportesRepo) GetVentasPorPeriodo(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, agrupacion string, sucursalID *uuid.UUID) ([]dto.VentasPorPeriodoResponse, error) {
	// Map Spanish agrupacion to PostgreSQL date_trunc interval
	pgInterval := "day"
	goFormat := "2006-01-02"
	switch agrupacion {
	case "semana", "week":
		pgInterval = "week"
		goFormat = "2006-01-02"
	case "mes", "month":
		pgInterval = "month"
		goFormat = "2006-01"
	default:
		// dia / day — default
		pgInterval = "day"
		goFormat = "2006-01-02"
	}

	type row struct {
		Periodo  time.Time
		Total    decimal.Decimal
		Cantidad int64
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Table("ventas").
		Select(fmt.Sprintf("date_trunc('%s', created_at) AS periodo, COALESCE(SUM(total), 0) AS total, COUNT(*) AS cantidad", pgInterval)).
		Where("tenant_id = ? AND estado = 'completada' AND created_at >= ? AND created_at < ?",
			tenantID, desde, hasta)
	if sucursalID != nil {
		q = q.Where("sucursal_id = ?", *sucursalID)
	}
	err := q.Group("periodo").
		Order("periodo ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("GetVentasPorPeriodo: %w", err)
	}

	result := make([]dto.VentasPorPeriodoResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.VentasPorPeriodoResponse{
			Periodo:  row.Periodo.Format(goFormat),
			Total:    row.Total,
			Cantidad: row.Cantidad,
		})
	}
	return result, nil
}

// GetVentasPorCajero returns aggregated sales metrics grouped by cashier (usuario_id).
func (r *reportesRepo) GetVentasPorCajero(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) ([]dto.ReporteCajeroResponse, error) {
	type row struct {
		UsuarioID       uuid.UUID
		NombreCajero    string
		TotalVentas     decimal.Decimal
		CantidadVentas  int64
		TotalDescuentos decimal.Decimal
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Table("ventas v").
		Select(`v.usuario_id,
			u.nombre AS nombre_cajero,
			COALESCE(SUM(v.total), 0) AS total_ventas,
			COUNT(*) AS cantidad_ventas,
			COALESCE(SUM(v.descuento_total), 0) AS total_descuentos`).
		Joins("JOIN usuarios u ON u.id = v.usuario_id AND u.tenant_id = ?", tenantID).
		Where("v.tenant_id = ? AND v.estado = 'completada' AND v.created_at >= ? AND v.created_at < ?",
			tenantID, desde, hasta)
	if sucursalID != nil {
		q = q.Where("v.sucursal_id = ?", *sucursalID)
	}
	err := q.Group("v.usuario_id, u.nombre").
		Order("total_ventas DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("GetVentasPorCajero: %w", err)
	}

	// Count anulaciones per cashier in the same period
	type anulRow struct {
		UsuarioID          uuid.UUID
		CantidadAnulaciones int64
	}
	var anulRows []anulRow

	aq := r.db.WithContext(ctx).
		Table("ventas").
		Select("usuario_id, COUNT(*) AS cantidad_anulaciones").
		Where("tenant_id = ? AND estado = 'anulada' AND created_at >= ? AND created_at < ?",
			tenantID, desde, hasta)
	if sucursalID != nil {
		aq = aq.Where("sucursal_id = ?", *sucursalID)
	}
	err = aq.Group("usuario_id").
		Scan(&anulRows).Error
	if err != nil {
		return nil, fmt.Errorf("GetVentasPorCajero anulaciones: %w", err)
	}

	anulMap := make(map[uuid.UUID]int64, len(anulRows))
	for _, a := range anulRows {
		anulMap[a.UsuarioID] = a.CantidadAnulaciones
	}

	result := make([]dto.ReporteCajeroResponse, 0, len(rows))
	for _, row := range rows {
		ticketPromedio := decimal.Zero
		if row.CantidadVentas > 0 {
			ticketPromedio = row.TotalVentas.Div(decimal.NewFromInt(row.CantidadVentas)).RoundBank(2)
		}
		result = append(result, dto.ReporteCajeroResponse{
			UsuarioID:           row.UsuarioID.String(),
			NombreCajero:        row.NombreCajero,
			TotalVentas:         row.TotalVentas,
			CantidadVentas:      row.CantidadVentas,
			TicketPromedio:      ticketPromedio,
			TotalDescuentos:     row.TotalDescuentos,
			CantidadAnulaciones: anulMap[row.UsuarioID],
		})
	}
	return result, nil
}

// GetReporteTurnos returns cash sessions (shifts) with aggregated venta totals and desvio info.
func (r *reportesRepo) GetReporteTurnos(ctx context.Context, tenantID uuid.UUID, desde, hasta time.Time, sucursalID *uuid.UUID) ([]dto.ReporteTurnoResponse, error) {
	type row struct {
		SesionID            uuid.UUID
		CajeroNombre        string
		FechaApertura       time.Time
		FechaCierre         *time.Time
		TotalVentas         decimal.Decimal
		CantidadVentas      int64
		Desvio              *decimal.Decimal
		ClasificacionDesvio *string
	}
	var rows []row

	q := r.db.WithContext(ctx).
		Table("sesiones_caja sc").
		Select(`sc.id AS sesion_id,
			u.nombre AS cajero_nombre,
			sc.opened_at AS fecha_apertura,
			sc.closed_at AS fecha_cierre,
			COALESCE(SUM(CASE WHEN v.estado = 'completada' THEN v.total ELSE 0 END), 0) AS total_ventas,
			COUNT(CASE WHEN v.estado = 'completada' THEN 1 END) AS cantidad_ventas,
			sc.desvio,
			sc.clasificacion_desvio`).
		Joins("JOIN usuarios u ON u.id = sc.usuario_id AND u.tenant_id = ?", tenantID).
		Joins("LEFT JOIN ventas v ON v.sesion_caja_id = sc.id AND v.tenant_id = ?", tenantID).
		Where("sc.tenant_id = ? AND sc.opened_at >= ? AND sc.opened_at < ?",
			tenantID, desde, hasta)
	if sucursalID != nil {
		q = q.Where("sc.sucursal_id = ?", *sucursalID)
	}
	err := q.Group("sc.id, u.nombre, sc.opened_at, sc.closed_at, sc.desvio, sc.clasificacion_desvio").
		Order("sc.opened_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("GetReporteTurnos: %w", err)
	}

	result := make([]dto.ReporteTurnoResponse, 0, len(rows))
	for _, row := range rows {
		var fechaCierre *string
		if row.FechaCierre != nil {
			s := row.FechaCierre.Format(time.RFC3339)
			fechaCierre = &s
		}

		desvio := decimal.Zero
		if row.Desvio != nil {
			desvio = *row.Desvio
		}

		clasificacion := "normal"
		if row.ClasificacionDesvio != nil {
			clasificacion = *row.ClasificacionDesvio
		}

		result = append(result, dto.ReporteTurnoResponse{
			SesionID:            row.SesionID.String(),
			CajeroNombre:        row.CajeroNombre,
			FechaApertura:       row.FechaApertura.Format(time.RFC3339),
			FechaCierre:         fechaCierre,
			TotalVentas:         row.TotalVentas,
			CantidadVentas:      row.CantidadVentas,
			Desvio:              desvio,
			DesvioClasificacion: clasificacion,
		})
	}
	return result, nil
}
