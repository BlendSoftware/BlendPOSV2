package handler

import (
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var tenantValidate = validator.New()

// TenantsHandler exposes tenant self-service and superadmin endpoints.
type TenantsHandler struct {
	svc service.TenantService
}

func NewTenantsHandler(svc service.TenantService) *TenantsHandler {
	return &TenantsHandler{svc: svc}
}

// Register godoc
// @Summary Registrar nuevo tenant
// @Tags    tenants
// @Accept  json
// @Produce json
// @Param   body body dto.RegisterTenantRequest true "Datos de registro"
// @Success 201 {object} dto.RegisterTenantResponse
// @Router  /v1/public/register [post]
func (h *TenantsHandler) Register(c *gin.Context) {
	var req dto.RegisterTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	if err := tenantValidate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	resp, err := h.svc.Registrar(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// ObtenerTenantActual godoc
// @Summary Obtener info del tenant actual
// @Tags    tenants
// @Produce json
// @Success 200 {object} dto.TenantResponse
// @Router  /v1/tenant/me [get]
func (h *TenantsHandler) ObtenerTenantActual(c *gin.Context) {
	resp, err := h.svc.ObtenerActual(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ActualizarTenantActual godoc
// @Summary Actualizar datos del tenant actual
// @Tags    tenants
// @Accept  json
// @Produce json
// @Param   body body dto.ActualizarTenantRequest true "Datos a actualizar"
// @Success 200 {object} dto.TenantResponse
// @Router  /v1/tenant/me [put]
func (h *TenantsHandler) ActualizarTenantActual(c *gin.Context) {
	var req dto.ActualizarTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	resp, err := h.svc.ActualizarActual(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ObtenerPlanActual godoc
// @Summary Obtener plan del tenant actual
// @Tags    tenants
// @Produce json
// @Success 200 {object} dto.PlanResponse
// @Router  /v1/tenant/plan [get]
func (h *TenantsHandler) ObtenerPlanActual(c *gin.Context) {
	resp, err := h.svc.GetPlanActual(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, apierror.New("plan no configurado"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListarPlanes godoc
// @Summary Listar planes disponibles (público)
// @Tags    tenants
// @Produce json
// @Success 200 {array} dto.PlanResponse
// @Router  /v1/public/planes [get]
func (h *TenantsHandler) ListarPlanes(c *gin.Context) {
	resp, err := h.svc.ListarPlanes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ── Superadmin endpoints ──────────────────────────────────────────────────────

// ListarTodos godoc
// @Summary Listar todos los tenants (superadmin)
// @Tags    superadmin
// @Produce json
// @Success 200 {array} dto.SuperadminTenantListItem
// @Router  /v1/superadmin/tenants [get]
func (h *TenantsHandler) ListarTodos(c *gin.Context) {
	items, err := h.svc.ListarTodos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, items)
}

// CambiarPlan godoc
// @Summary Cambiar plan de un tenant (superadmin)
// @Tags    superadmin
// @Accept  json
// @Produce json
// @Param   id   path string                 true "Tenant ID"
// @Param   body body dto.CambiarPlanRequest true "Plan ID"
// @Success 200 {object} dto.TenantResponse
// @Router  /v1/superadmin/tenants/:id/plan [put]
func (h *TenantsHandler) CambiarPlan(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("tenant ID inválido"))
		return
	}
	var req dto.CambiarPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("plan ID inválido"))
		return
	}
	resp, err := h.svc.CambiarPlan(c.Request.Context(), tenantID, planID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ToggleActivo godoc
// @Summary Activar/desactivar tenant (superadmin)
// @Tags    superadmin
// @Accept  json
// @Produce json
// @Param   id   path string                  true "Tenant ID"
// @Param   body body dto.ToggleTenantRequest true "Estado"
// @Success 200 {object} dto.TenantResponse
// @Router  /v1/superadmin/tenants/:id [put]
func (h *TenantsHandler) ToggleActivo(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("tenant ID inválido"))
		return
	}
	var req dto.ToggleTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New(err.Error()))
		return
	}
	resp, err := h.svc.ToggleActivo(c.Request.Context(), tenantID, req.Activo)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ObtenerMetricas godoc
// @Summary Métricas globales (superadmin)
// @Tags    superadmin
// @Produce json
// @Success 200 {object} dto.SuperadminMetricsResponse
// @Router  /v1/superadmin/metrics [get]
func (h *TenantsHandler) ObtenerMetricas(c *gin.Context) {
	resp, err := h.svc.ObtenerMetricas(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
