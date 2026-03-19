package handler

// reportes.go
// Analytics endpoints (T5.1+T5.2) — reads from the read replica when available.
// Uses explicit tenant_id WHERE clause via service/repo layer because the
// replica does NOT go through TenantMiddleware's set_config (RLS is not active).

import (
	"net/http"
	"strconv"
	"time"

	"blendpos/internal/apierror"
	"blendpos/internal/middleware"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseSucursalID reads the optional "sucursal_id" query param.
// Returns nil when absent or empty (consolidated view).
func parseSucursalID(c *gin.Context) *uuid.UUID {
	raw := c.Query("sucursal_id")
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// ReportesHandler exposes analytics endpoints.
type ReportesHandler struct {
	svc service.ReportesService
}

// NewReportesHandler creates a handler wired to the reportes service.
func NewReportesHandler(svc service.ReportesService) *ReportesHandler {
	return &ReportesHandler{svc: svc}
}

// defaultDateRange returns (first day of current month, today) as YYYY-MM-DD strings.
func defaultDateRange() (string, string) {
	now := time.Now()
	desde := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	hasta := now.Format("2006-01-02")
	return desde, hasta
}

// GetResumen handles GET /v1/reportes/resumen?desde=&hasta=
// @Summary      Resumen de ventas
// @Description  Retorna total vendido, cantidad de ventas y ticket promedio para el período.
// @Tags         reportes
// @Produce      json
// @Security     BearerAuth
// @Param        desde query string false "Fecha inicio YYYY-MM-DD (default: 1er día del mes)"
// @Param        hasta query string false "Fecha fin YYYY-MM-DD (default: hoy)"
// @Success      200 {object} dto.VentasResumenResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/reportes/resumen [get]
func (h *ReportesHandler) GetResumen(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	desdeDefault, hastaDefault := defaultDateRange()
	desde := c.DefaultQuery("desde", desdeDefault)
	hasta := c.DefaultQuery("hasta", hastaDefault)

	sucursalID := parseSucursalID(c)
	resp, err := h.svc.GetVentasResumen(c.Request.Context(), tenantID, desde, hasta, sucursalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetTopProductos handles GET /v1/reportes/top-productos?desde=&hasta=&limit=10
// @Summary      Top productos vendidos
// @Description  Retorna los productos más vendidos por cantidad en el período.
// @Tags         reportes
// @Produce      json
// @Security     BearerAuth
// @Param        desde query string false "Fecha inicio YYYY-MM-DD"
// @Param        hasta query string false "Fecha fin YYYY-MM-DD"
// @Param        limit query int    false "Cantidad de productos (default 10, max 100)"
// @Success      200 {array}  dto.TopProductoResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/reportes/top-productos [get]
func (h *ReportesHandler) GetTopProductos(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	desdeDefault, hastaDefault := defaultDateRange()
	desde := c.DefaultQuery("desde", desdeDefault)
	hasta := c.DefaultQuery("hasta", hastaDefault)

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	sucursalID := parseSucursalID(c)
	resp, err := h.svc.GetTopProductos(c.Request.Context(), tenantID, desde, hasta, limit, sucursalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetMediosPago handles GET /v1/reportes/medios-pago?desde=&hasta=
// @Summary      Ventas por medio de pago
// @Description  Desglosa ventas por método de pago (efectivo, débito, crédito, transferencia).
// @Tags         reportes
// @Produce      json
// @Security     BearerAuth
// @Param        desde query string false "Fecha inicio YYYY-MM-DD"
// @Param        hasta query string false "Fecha fin YYYY-MM-DD"
// @Success      200 {array}  dto.VentasPorMedioPagoResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/reportes/medios-pago [get]
func (h *ReportesHandler) GetMediosPago(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	desdeDefault, hastaDefault := defaultDateRange()
	desde := c.DefaultQuery("desde", desdeDefault)
	hasta := c.DefaultQuery("hasta", hastaDefault)

	sucursalID := parseSucursalID(c)
	resp, err := h.svc.GetVentasPorMedioPago(c.Request.Context(), tenantID, desde, hasta, sucursalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetVentasPeriodo handles GET /v1/reportes/ventas-periodo?desde=&hasta=&agrupacion=dia
// @Summary      Ventas por período
// @Description  Agrupa ventas por día, semana o mes para visualización de tendencias.
// @Tags         reportes
// @Produce      json
// @Security     BearerAuth
// @Param        desde      query string false "Fecha inicio YYYY-MM-DD"
// @Param        hasta      query string false "Fecha fin YYYY-MM-DD"
// @Param        agrupacion query string false "dia|semana|mes (default: dia)"
// @Success      200 {array}  dto.VentasPorPeriodoResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/reportes/ventas-periodo [get]
func (h *ReportesHandler) GetVentasPeriodo(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	desdeDefault, hastaDefault := defaultDateRange()
	desde := c.DefaultQuery("desde", desdeDefault)
	hasta := c.DefaultQuery("hasta", hastaDefault)
	agrupacion := c.DefaultQuery("agrupacion", "dia")

	sucursalID := parseSucursalID(c)
	resp, err := h.svc.GetVentasPorPeriodo(c.Request.Context(), tenantID, desde, hasta, agrupacion, sucursalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetCajeros handles GET /v1/reportes/cajeros?desde=&hasta=
// @Summary      Ventas por cajero
// @Description  Retorna métricas de ventas agrupadas por cajero para el período.
// @Tags         reportes
// @Produce      json
// @Security     BearerAuth
// @Param        desde query string false "Fecha inicio YYYY-MM-DD (default: 1er día del mes)"
// @Param        hasta query string false "Fecha fin YYYY-MM-DD (default: hoy)"
// @Success      200 {array}  dto.ReporteCajeroResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/reportes/cajeros [get]
func (h *ReportesHandler) GetCajeros(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	desdeDefault, hastaDefault := defaultDateRange()
	desde := c.DefaultQuery("desde", desdeDefault)
	hasta := c.DefaultQuery("hasta", hastaDefault)

	sucursalID := parseSucursalID(c)
	resp, err := h.svc.GetVentasPorCajero(c.Request.Context(), tenantID, desde, hasta, sucursalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetTurnos handles GET /v1/reportes/turnos?desde=&hasta=
// @Summary      Reporte de turnos (sesiones de caja)
// @Description  Retorna sesiones de caja con totales de venta y desvío para el período.
// @Tags         reportes
// @Produce      json
// @Security     BearerAuth
// @Param        desde query string false "Fecha inicio YYYY-MM-DD (default: 1er día del mes)"
// @Param        hasta query string false "Fecha fin YYYY-MM-DD (default: hoy)"
// @Success      200 {array}  dto.ReporteTurnoResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/reportes/turnos [get]
func (h *ReportesHandler) GetTurnos(c *gin.Context) {
	tenantID, err := middleware.TenantIDFromContext(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("tenant context missing"))
		return
	}

	desdeDefault, hastaDefault := defaultDateRange()
	desde := c.DefaultQuery("desde", desdeDefault)
	hasta := c.DefaultQuery("hasta", hastaDefault)

	sucursalID := parseSucursalID(c)
	resp, err := h.svc.GetReporteTurnos(c.Request.Context(), tenantID, desde, hasta, sucursalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
