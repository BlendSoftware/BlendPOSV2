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

type TransferenciasHandler struct{ svc service.TransferenciaService }

func NewTransferenciasHandler(svc service.TransferenciaService) *TransferenciasHandler {
	return &TransferenciasHandler{svc: svc}
}

// Crear POST /v1/transferencias
func (h *TransferenciasHandler) Crear(c *gin.Context) {
	var req dto.CrearTransferenciaRequest
	if !bindAndValidate(c, &req) {
		return
	}
	claims := middleware.GetClaims(c)
	usuarioID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("usuario_id inválido en token"))
		return
	}
	resp, err := h.svc.CrearTransferencia(c.Request.Context(), usuarioID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	id, _ := uuid.Parse(resp.ID)
	middleware.AuditLog(c, "create", "transferencia_stock", &id, map[string]interface{}{
		"origen":  req.SucursalOrigenID,
		"destino": req.SucursalDestinoID,
		"items":   len(req.Items),
	})
	c.JSON(http.StatusCreated, resp)
}

// Listar GET /v1/transferencias
func (h *TransferenciasHandler) Listar(c *gin.Context) {
	estado := c.Query("estado")
	resp, err := h.svc.ListarTransferencias(c.Request.Context(), estado)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar transferencias"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ObtenerPorID GET /v1/transferencias/:id
func (h *TransferenciasHandler) ObtenerPorID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID inválido"))
		return
	}
	resp, err := h.svc.ObtenerTransferencia(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Completar POST /v1/transferencias/:id/completar
func (h *TransferenciasHandler) Completar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID inválido"))
		return
	}
	claims := middleware.GetClaims(c)
	usuarioID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, apierror.New("usuario_id inválido en token"))
		return
	}
	resp, err := h.svc.CompletarTransferencia(c.Request.Context(), id, usuarioID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "complete", "transferencia_stock", &id, nil)
	c.JSON(http.StatusOK, resp)
}

// Rechazar POST /v1/transferencias/:id/rechazar
func (h *TransferenciasHandler) Rechazar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID inválido"))
		return
	}
	if err := h.svc.RechazarTransferencia(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "reject", "transferencia_stock", &id, nil)
	c.Status(http.StatusNoContent)
}

// Cancelar POST /v1/transferencias/:id/cancelar
func (h *TransferenciasHandler) Cancelar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID inválido"))
		return
	}
	if err := h.svc.CancelarTransferencia(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "cancel", "transferencia_stock", &id, nil)
	c.Status(http.StatusNoContent)
}

// ListarStockSucursal GET /v1/stock-sucursal?sucursal_id=
func (h *TransferenciasHandler) ListarStockSucursal(c *gin.Context) {
	// Resolve sucursal: explicit query param > header (SucursalMiddleware).
	sucursalPtr := parseSucursalID(c)
	if sucursalPtr == nil {
		c.JSON(http.StatusBadRequest, apierror.New("sucursal_id es requerido"))
		return
	}
	sucursalID := *sucursalPtr
	resp, err := h.svc.ListarStockSucursal(c.Request.Context(), sucursalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar stock por sucursal"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// AjustarStockSucursal POST /v1/stock-sucursal/ajustar
func (h *TransferenciasHandler) AjustarStockSucursal(c *gin.Context) {
	var req dto.AjustarStockSucursalRequest
	if !bindAndValidate(c, &req) {
		return
	}
	if err := h.svc.AjustarStockSucursal(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stock ajustado correctamente"})
}

// GetAlertasBySucursal GET /v1/stock-sucursal/alertas?sucursal_id=
func (h *TransferenciasHandler) GetAlertasBySucursal(c *gin.Context) {
	// Resolve sucursal: explicit query param > header (SucursalMiddleware).
	sucursalPtr := parseSucursalID(c)
	if sucursalPtr == nil {
		c.JSON(http.StatusBadRequest, apierror.New("sucursal_id es requerido"))
		return
	}
	sucursalID := *sucursalPtr
	resp, err := h.svc.GetAlertasBySucursal(c.Request.Context(), sucursalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al obtener alertas de stock"))
		return
	}
	c.JSON(http.StatusOK, resp)
}
