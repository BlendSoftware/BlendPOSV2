package handler

import (
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/middleware"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VentasHandler struct{ svc service.VentaService }

func NewVentasHandler(svc service.VentaService) *VentasHandler { return &VentasHandler{svc: svc} }

// RegistrarVenta godoc
// @Summary      Registrar una nueva venta
// @Description  Crea una venta ACID: descuenta stock, crea movimientos de caja y despacha facturación AFIP asíncrona.
// @Tags         ventas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.RegistrarVentaRequest true "Detalle de la venta"
// @Success      201  {object} dto.VentaResponse
// @Failure      400  {object} apierror.APIError
// @Router       /v1/ventas [post]
func (h *VentasHandler) RegistrarVenta(c *gin.Context) {
	var req dto.RegistrarVentaRequest
	if !bindAndValidate(c, &req) {
		return
	}
	claims := middleware.GetClaims(c)
	usuarioID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Token malformado: user_id inválido"))
		return
	}

	resp, err2 := h.svc.RegistrarVenta(c.Request.Context(), usuarioID, req)
	if err2 != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err2.Error()))
		return
	}
	ventaID, _ := uuid.Parse(resp.ID)
	middleware.AuditLog(c, "create", "venta", &ventaID, map[string]interface{}{"total": resp.Total, "items": len(req.Items)})
	c.JSON(http.StatusCreated, resp)
}

// AnularVenta godoc
// @Summary      Anular venta
// @Description  Anula una venta: restaura stock y genera movimientos de caja inversos.
// @Tags         ventas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path     string               true "UUID de la venta"
// @Param        body body     dto.AnularVentaRequest true "Motivo de anulación"
// @Success      204
// @Failure      400  {object} apierror.APIError
// @Router       /v1/ventas/{id} [delete]
func (h *VentasHandler) AnularVenta(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.AnularVentaRequest
	if !bindAndValidate(c, &req) {
		return
	}
	if err := h.svc.AnularVenta(c.Request.Context(), id, req.Motivo); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "anular", "venta", &id, map[string]interface{}{"motivo": req.Motivo})
	c.Status(http.StatusNoContent)
}

// ListarVentas godoc
// @Summary      Listar ventas
// @Description  Retorna lista paginada de ventas filtrada por fecha y estado.
// @Tags         ventas
// @Produce      json
// @Security     BearerAuth
// @Param        fecha  query string false "Fecha YYYY-MM-DD (default: hoy)"
// @Param        estado query string false "completada | anulada | all"
// @Param        page   query int    false "Página (default 1)"
// @Param        limit  query int    false "Registros por página (default 50)"
// @Success      200    {object} dto.VentaListResponse
// @Failure      400    {object} apierror.APIError
// @Router       /v1/ventas [get]
// ListarVentas returns a paginated, filtered list of sales.
func (h *VentasHandler) ListarVentas(c *gin.Context) {
	var filter dto.VentaFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	// When no explicit sucursal_id query param, fall back to header from global selector.
	if filter.SucursalID == "" {
		if sid := parseSucursalID(c); sid != nil {
			filter.SucursalID = sid.String()
		}
	}
	resp, err := h.svc.ListVentas(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar ventas"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// SyncBatch godoc
// @Summary      Sincronizar ventas offline (multi-tenant)
// @Description  Procesa un lote de ventas creadas offline. Idempotente por (tenant_id, offline_id).
//               tenant_id se inyecta desde JWT — cualquier tenant_id en el body es ignorado.
//               Aplica auto-compensación de stock (offline-first: nunca rechaza).
// @Tags         ventas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.SyncBatchRequest true "Lote de ventas"
// @Success      200  {object} dto.SyncBatchResponse
// @Failure      400  {object} apierror.APIError
// @Router       /v1/ventas/sync-batch [post]
func (h *VentasHandler) SyncBatch(c *gin.Context) {
	var req dto.SyncBatchRequest
	if !bindAndValidate(c, &req) {
		return
	}
	claims := middleware.GetClaims(c)
	usuarioID, parseErr := uuid.Parse(claims.UserID)
	if parseErr != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Token malformado: user_id inválido"))
		return
	}

	resp, err := h.svc.SyncBatch(c.Request.Context(), usuarioID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}

	// Audit log: batch sync summary
	middleware.AuditLog(c, "sync-batch", "venta", nil, map[string]interface{}{
		"synced":     len(resp.SyncedIDs),
		"duplicated": len(resp.DuplicatedIDs),
		"failed":     len(resp.FailedIDs),
		"device_id":  req.DeviceID,
	})

	c.JSON(http.StatusOK, resp)
}
