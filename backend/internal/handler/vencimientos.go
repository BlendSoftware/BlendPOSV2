package handler

import (
	"net/http"
	"strconv"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VencimientosHandler struct{ svc service.LoteService }

func NewVencimientosHandler(svc service.LoteService) *VencimientosHandler {
	return &VencimientosHandler{svc: svc}
}

// CrearLote POST /v1/lotes
func (h *VencimientosHandler) CrearLote(c *gin.Context) {
	var req dto.CrearLoteRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.CrearLote(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// ListarLotes GET /v1/lotes?producto_id=
func (h *VencimientosHandler) ListarLotes(c *gin.Context) {
	productoIDStr := c.Query("producto_id")
	if productoIDStr == "" {
		c.JSON(http.StatusBadRequest, apierror.New("producto_id es requerido"))
		return
	}
	productoID, err := uuid.Parse(productoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("producto_id inválido"))
		return
	}
	resp, err := h.svc.ListarLotes(c.Request.Context(), productoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar lotes"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// EliminarLote DELETE /v1/lotes/:id
func (h *VencimientosHandler) EliminarLote(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("id inválido"))
		return
	}
	if err := h.svc.EliminarLote(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, apierror.New(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

// ObtenerAlertasVencimiento GET /v1/vencimientos/alertas?dias=7
func (h *VencimientosHandler) ObtenerAlertasVencimiento(c *gin.Context) {
	dias := 7
	if d := c.Query("dias"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			dias = parsed
		}
	}
	resp, err := h.svc.ObtenerAlertasVencimiento(c.Request.Context(), dias)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al obtener alertas de vencimiento"))
		return
	}
	c.JSON(http.StatusOK, resp)
}
