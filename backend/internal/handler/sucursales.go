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

type SucursalesHandler struct{ svc service.SucursalService }

func NewSucursalesHandler(svc service.SucursalService) *SucursalesHandler {
	return &SucursalesHandler{svc: svc}
}

// Crear godoc
// @Summary      Crear sucursal
// @Description  Crea una nueva sucursal (branch) para el tenant.
// @Tags         sucursales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.CrearSucursalRequest true "Datos de la sucursal"
// @Success      201  {object} dto.SucursalResponse
// @Failure      400  {object} apierror.APIError
// @Router       /v1/sucursales [post]
func (h *SucursalesHandler) Crear(c *gin.Context) {
	var req dto.CrearSucursalRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.Crear(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	id, _ := uuid.Parse(resp.ID)
	middleware.AuditLog(c, "create", "sucursal", &id, map[string]interface{}{"nombre": req.Nombre})
	c.JSON(http.StatusCreated, resp)
}

// Listar godoc
// @Summary      Listar sucursales
// @Description  Lista las sucursales del tenant. Por defecto solo activas.
// @Tags         sucursales
// @Produce      json
// @Security     BearerAuth
// @Param        incluir_inactivas query bool false "Incluir sucursales inactivas"
// @Success      200 {object} dto.SucursalListResponse
// @Router       /v1/sucursales [get]
func (h *SucursalesHandler) Listar(c *gin.Context) {
	incluirInactivas := c.Query("incluir_inactivas") == "true"
	resp, err := h.svc.Listar(c.Request.Context(), incluirInactivas)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar sucursales"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ObtenerPorID godoc
// @Summary      Obtener detalle de sucursal
// @Description  Retorna una sucursal por su UUID.
// @Tags         sucursales
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "UUID de la sucursal"
// @Success      200 {object} dto.SucursalResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/sucursales/{id} [get]
func (h *SucursalesHandler) ObtenerPorID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	resp, err := h.svc.ObtenerPorID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Actualizar godoc
// @Summary      Actualizar sucursal
// @Description  Actualiza datos de la sucursal (nombre, direccion, telefono, activa).
// @Tags         sucursales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path string true "UUID de la sucursal"
// @Param        body body dto.UpdateSucursalRequest true "Campos a actualizar"
// @Success      200 {object} dto.SucursalResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/sucursales/{id} [put]
func (h *SucursalesHandler) Actualizar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.UpdateSucursalRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.Actualizar(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "update", "sucursal", &id, nil)
	c.JSON(http.StatusOK, resp)
}

// Desactivar godoc
// @Summary      Desactivar sucursal (soft delete)
// @Description  Marca la sucursal como inactiva (activa=false).
// @Tags         sucursales
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "UUID de la sucursal"
// @Success      204
// @Failure      400 {object} apierror.APIError
// @Router       /v1/sucursales/{id} [delete]
func (h *SucursalesHandler) Desactivar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	activa := false
	_, err = h.svc.Actualizar(c.Request.Context(), id, dto.UpdateSucursalRequest{Activa: &activa})
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "delete", "sucursal", &id, nil)
	c.Status(http.StatusNoContent)
}
