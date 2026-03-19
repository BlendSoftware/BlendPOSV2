package handler

import (
	"net/http"
	"strconv"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/middleware"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClientesHandler struct{ svc service.ClienteService }

func NewClientesHandler(svc service.ClienteService) *ClientesHandler {
	return &ClientesHandler{svc: svc}
}

// Crear godoc
// @Summary      Crear cliente
// @Description  Crea un nuevo cliente con cuenta corriente (fiado).
// @Tags         clientes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.CrearClienteRequest true "Datos del cliente"
// @Success      201  {object} dto.ClienteResponse
// @Failure      400  {object} apierror.APIError
// @Router       /v1/clientes [post]
func (h *ClientesHandler) Crear(c *gin.Context) {
	var req dto.CrearClienteRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.Crear(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	id, _ := uuid.Parse(resp.ID)
	middleware.AuditLog(c, "create", "cliente", &id, map[string]interface{}{"nombre": req.Nombre})
	c.JSON(http.StatusCreated, resp)
}

// Listar godoc
// @Summary      Listar clientes
// @Description  Lista clientes activos con búsqueda opcional por nombre.
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Param        search query string false "Buscar por nombre"
// @Param        page   query int    false "Página (default 1)"
// @Param        limit  query int    false "Registros por página (default 50)"
// @Success      200    {object} dto.ClienteListResponse
// @Router       /v1/clientes [get]
func (h *ClientesHandler) Listar(c *gin.Context) {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	resp, err := h.svc.Listar(c.Request.Context(), search, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar clientes"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ObtenerPorID godoc
// @Summary      Obtener detalle de cliente
// @Description  Retorna el cliente con su saldo deudor actual.
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "UUID del cliente"
// @Success      200 {object} dto.ClienteResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/clientes/{id} [get]
func (h *ClientesHandler) ObtenerPorID(c *gin.Context) {
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
// @Summary      Actualizar cliente
// @Description  Actualiza datos del cliente (nombre, teléfono, límite crédito, etc.)
// @Tags         clientes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path string true "UUID del cliente"
// @Param        body body dto.UpdateClienteRequest true "Campos a actualizar"
// @Success      200 {object} dto.ClienteResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/clientes/{id} [put]
func (h *ClientesHandler) Actualizar(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.UpdateClienteRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.Actualizar(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "update", "cliente", &id, nil)
	c.JSON(http.StatusOK, resp)
}

// RegistrarPago godoc
// @Summary      Registrar pago de cuenta corriente
// @Description  Registra un pago que reduce el saldo deudor del cliente.
// @Tags         clientes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path string true "UUID del cliente"
// @Param        body body dto.RegistrarPagoClienteRequest true "Monto y descripción"
// @Success      201 {object} dto.MovimientoCuentaResponse
// @Failure      400 {object} apierror.APIError
// @Router       /v1/clientes/{id}/pago [post]
func (h *ClientesHandler) RegistrarPago(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	var req dto.RegistrarPagoClienteRequest
	if !bindAndValidate(c, &req) {
		return
	}
	resp, err := h.svc.RegistrarPago(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	middleware.AuditLog(c, "pago", "cliente", &id, map[string]interface{}{"monto": req.Monto.StringFixed(2)})
	c.JSON(http.StatusCreated, resp)
}

// ListarMovimientos godoc
// @Summary      Historial de movimientos de cuenta corriente
// @Description  Lista paginada de movimientos (cargos, pagos, ajustes) del cliente.
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string true  "UUID del cliente"
// @Param        page  query int    false "Página (default 1)"
// @Param        limit query int    false "Registros por página (default 50)"
// @Success      200 {object} dto.MovimientosListResponse
// @Router       /v1/clientes/{id}/movimientos [get]
func (h *ClientesHandler) ListarMovimientos(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("ID invalido"))
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	resp, err := h.svc.GetMovimientos(c.Request.Context(), id, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar movimientos"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListarDeudores godoc
// @Summary      Listar deudores
// @Description  Lista todos los clientes con saldo deudor > 0, ordenados por saldo descendente.
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.ListDeudoresResponse
// @Router       /v1/clientes/deudores [get]
func (h *ClientesHandler) ListarDeudores(c *gin.Context) {
	resp, err := h.svc.GetDeudores(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New("Error al listar deudores"))
		return
	}
	c.JSON(http.StatusOK, resp)
}
