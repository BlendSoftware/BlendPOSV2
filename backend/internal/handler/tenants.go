package handler

import (
	"net/http"

	"blendpos/internal/apierror"
	"blendpos/internal/dto"
	"blendpos/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
	if !bindAndValidate(c, &req) {
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
// @Summary Listar todos los tenants con paginación (superadmin)
// @Tags    superadmin
// @Produce json
// @Param   page      query int    false "Page number"     default(1)
// @Param   page_size query int    false "Items per page"  default(20)
// @Param   search    query string false "Search by name or slug"
// @Param   status    query string false "Filter: active, inactive, all" default(all)
// @Param   plan_id   query string false "Filter by plan UUID"
// @Success 200 {object} dto.TenantListResponse
// @Router  /v1/superadmin/tenants [get]
func (h *TenantsHandler) ListarTodos(c *gin.Context) {
	var req dto.TenantListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("parámetros de consulta inválidos"))
		return
	}
	resp, err := h.svc.ListarTodos(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ObtenerTenantDetalle godoc
// @Summary Detalle de un tenant con métricas (superadmin)
// @Tags    superadmin
// @Produce json
// @Param   id path string true "Tenant ID"
// @Success 200 {object} dto.SuperadminTenantListItem
// @Router  /v1/superadmin/tenants/:id [get]
func (h *TenantsHandler) ObtenerTenantDetalle(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, apierror.New("tenant ID inválido"))
		return
	}
	resp, err := h.svc.ObtenerTenantDetalle(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, apierror.New(err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
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

// ListarPresets godoc
// @Summary Listar resumen de presets de negocio (público)
// @Tags    tenants
// @Produce json
// @Success 200 {array} dto.PresetResponse
// @Router  /v1/public/presets [get]
func (h *TenantsHandler) ListarPresets(c *gin.Context) {
	c.JSON(http.StatusOK, service.GetAllPresetSummaries())
}

// ObtenerPreset godoc
// @Summary Obtener preset por tipo de negocio (público)
// @Tags    tenants
// @Produce json
// @Param   tipo path string true "Tipo de negocio"
// @Success 200 {object} dto.PresetResponse
// @Router  /v1/public/presets/:tipo [get]
func (h *TenantsHandler) ObtenerPreset(c *gin.Context) {
	tipo := c.Param("tipo")
	resp, err := service.GetPresetInfo(tipo)
	if err != nil {
		c.JSON(http.StatusNotFound, apierror.New(err.Error()))
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
