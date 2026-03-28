package handler

// venta_reporte.go
// Analytics endpoint that reads from the PostgreSQL read replica (F1-9).
// Uses explicit tenant_id WHERE clause — does NOT rely on RLS because the
// replica connection does not go through TenantMiddleware's set_config call.

import (
	"net/http"
	"time"

	"blendpos/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// VentaReporteHandler handles analytics queries against the read replica DB.
type VentaReporteHandler struct {
	dbRead *gorm.DB
}

// NewVentaReporteHandler creates a handler that queries the read replica.
// If the replica is not configured, dbRead is the primary DB (transparent fallback).
func NewVentaReporteHandler(dbRead *gorm.DB) *VentaReporteHandler {
	return &VentaReporteHandler{dbRead: dbRead}
}

// VentaReporteResponse is the PoC analytics payload for GET /v1/ventas/reporte.
type VentaReporteResponse struct {
	TotalVentas    int64           `json:"total_ventas"`
	MontoTotal     decimal.Decimal `json:"monto_total"`
	PromedioVenta  decimal.Decimal `json:"promedio_venta"`
	PeriodoDesde   string          `json:"periodo_desde"`
	PeriodoHasta   string          `json:"periodo_hasta"`
}

// GetReporte handles GET /v1/ventas/reporte?desde=YYYY-MM-DD&hasta=YYYY-MM-DD
// @Summary      Reporte de ventas (read replica)
// @Description  Retorna métricas agregadas de ventas para el período indicado. Lee de la réplica de lectura cuando está configurada.
// @Tags         ventas
// @Produce      json
// @Param        desde  query  string  false  "Fecha inicio (YYYY-MM-DD). Default: inicio del mes actual."
// @Param        hasta  query  string  false  "Fecha fin (YYYY-MM-DD). Default: hoy."
// @Success      200  {object}  VentaReporteResponse
// @Failure      400  {object}  apierror.APIError
// @Failure      401  {object}  apierror.APIError
// @Router       /v1/ventas/reporte [get]
func (h *VentaReporteHandler) GetReporte(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// Parse date range — default: current month
	now := time.Now()
	desdeStr := c.DefaultQuery("desde", time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
	hastaStr := c.DefaultQuery("hasta", now.Format("2006-01-02"))

	desde, err := time.Parse("2006-01-02", desdeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parámetro 'desde' inválido, use YYYY-MM-DD"})
		return
	}
	hasta, err := time.Parse("2006-01-02", hastaStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parámetro 'hasta' inválido, use YYYY-MM-DD"})
		return
	}
	// Exclusive upper bound: start of the next day (half-open interval)
	hastaFin := hasta.AddDate(0, 0, 1)

	type result struct {
		TotalVentas int64
		MontoTotal  decimal.Decimal
	}
	var r result

	// Explicit tenant_id filter — replica does not run RLS set_config
	err = h.dbRead.
		Table("ventas").
		Select("COUNT(*) AS total_ventas, COALESCE(SUM(total), 0) AS monto_total").
		Where("tenant_id = ? AND estado = 'completada' AND created_at >= ? AND created_at < ?",
			tenantID, desde, hastaFin).
		Scan(&r).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error al consultar reporte"})
		return
	}

	promedio := decimal.Zero
	if r.TotalVentas > 0 {
		promedio = r.MontoTotal.Div(decimal.NewFromInt(r.TotalVentas)).RoundBank(2)
	}

	c.JSON(http.StatusOK, VentaReporteResponse{
		TotalVentas:   r.TotalVentas,
		MontoTotal:    r.MontoTotal,
		PromedioVenta: promedio,
		PeriodoDesde:  desdeStr,
		PeriodoHasta:  hastaStr,
	})
}
