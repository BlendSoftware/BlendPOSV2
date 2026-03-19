package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"blendpos/internal/dto"
	"blendpos/internal/handler"
	"blendpos/internal/infra"
	"blendpos/internal/model"
	"blendpos/internal/repository"
	"blendpos/internal/service"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Stub ConfiguracionFiscal Repository ─────────────────────────────────────

type stubConfigFiscalRepo struct {
	configs map[uuid.UUID]*model.ConfiguracionFiscal // keyed by tenant_id
}

func newStubConfigFiscalRepo() *stubConfigFiscalRepo {
	return &stubConfigFiscalRepo{
		configs: make(map[uuid.UUID]*model.ConfiguracionFiscal),
	}
}

func (r *stubConfigFiscalRepo) Get(ctx context.Context) (*model.ConfiguracionFiscal, error) {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	cfg, ok := r.configs[tid]
	if !ok {
		return nil, nil
	}
	return cfg, nil
}

func (r *stubConfigFiscalRepo) Upsert(ctx context.Context, config *model.ConfiguracionFiscal) error {
	tid, err := tenantctx.FromContext(ctx)
	if err != nil {
		return err
	}
	config.TenantID = tid
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	r.configs[tid] = config
	return nil
}

// Ensure stub satisfies the interface at compile time.
var _ repository.ConfiguracionFiscalRepository = (*stubConfigFiscalRepo)(nil)

// ── Stub AFIP Client (no-op sidecar) ────────────────────────────────────────

type stubAFIPClient struct{}

func (s *stubAFIPClient) Facturar(_ context.Context, _ infra.AFIPPayload) (*infra.AFIPResponse, error) {
	return nil, nil
}
func (s *stubAFIPClient) GetSidecarURL() string    { return "" } // empty = skip sidecar
func (s *stubAFIPClient) GetInternalToken() string  { return "" }

var _ infra.AFIPClient = (*stubAFIPClient)(nil)

// ── Helpers ─────────────────────────────────────────────────────────────────

func newConfigFiscalTestService() (service.ConfiguracionFiscalService, *stubConfigFiscalRepo) {
	repo := newStubConfigFiscalRepo()
	svc := service.NewConfiguracionFiscalService(repo, &stubAFIPClient{})
	return svc, repo
}

func ctxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), tenantctx.Key, tenantID)
}

func setupConfigFiscalRouter(svc service.ConfiguracionFiscalService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewConfiguracionFiscalHandler(svc)
	// Inject tenant_id into context via a test middleware
	r.GET("/v1/configuracion/fiscal", func(c *gin.Context) {
		tid := c.GetHeader("X-Test-Tenant-ID")
		if tid != "" {
			parsed, _ := uuid.Parse(tid)
			c.Request = c.Request.WithContext(
				context.WithValue(c.Request.Context(), tenantctx.Key, parsed),
			)
		}
		c.Next()
	}, h.Obtener)
	r.PUT("/v1/configuracion/fiscal", func(c *gin.Context) {
		tid := c.GetHeader("X-Test-Tenant-ID")
		if tid != "" {
			parsed, _ := uuid.Parse(tid)
			c.Request = c.Request.WithContext(
				context.WithValue(c.Request.Context(), tenantctx.Key, parsed),
			)
		}
		c.Next()
	}, h.Actualizar)
	return r
}

func doGetConfigFiscal(t *testing.T, router *gin.Engine, tenantID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/v1/configuracion/fiscal", nil)
	req.Header.Set("X-Test-Tenant-ID", tenantID.String())
	router.ServeHTTP(w, req)
	return w
}

// ── Tests: Service Layer ────────────────────────────────────────────────────

func TestConfigFiscal_ObtenerConfiguracion_ReturnsTenantConfig(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()

	// Seed a config for tenant A
	crt := "base64cert"
	repo.configs[tenantA] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantA,
		CUITEmsior:      "20-12345678-9",
		RazonSocial:     "Kiosco A",
		CondicionFiscal: "Responsable Inscripto",
		PuntoDeVenta:    1,
		Modo:            "homologacion",
		CertificadoCrt:  &crt,
	}

	ctx := ctxWithTenant(tenantA)
	resp, err := svc.ObtenerConfiguracion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "20-12345678-9", resp.CUITEmsior)
	assert.Equal(t, "Kiosco A", resp.RazonSocial)
	assert.Equal(t, "homologacion", resp.Modo)
	assert.True(t, resp.TieneCertificadoCrt)
	assert.False(t, resp.TieneCertificadoKey)
}

func TestConfigFiscal_ObtenerConfiguracion_EmptyWhenNoConfig(t *testing.T) {
	svc, _ := newConfigFiscalTestService()
	tenantA := uuid.New()

	ctx := ctxWithTenant(tenantA)
	resp, err := svc.ObtenerConfiguracion(ctx)
	require.NoError(t, err)
	assert.Empty(t, resp.CUITEmsior)
	assert.Equal(t, 0, resp.PuntoDeVenta)
}

func TestConfigFiscal_ActualizarConfiguracion_SavesForTenant(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()
	ctx := ctxWithTenant(tenantA)

	err := svc.ActualizarConfiguracion(ctx, dto.ConfiguracionFiscalRequest{
		CUITEmsior:      "20-11111111-1",
		RazonSocial:     "Mi Kiosco",
		CondicionFiscal: "Monotributista",
		PuntoDeVenta:    2,
		Modo:            "produccion",
	})
	require.NoError(t, err)

	// Verify stored in the repo for the correct tenant
	cfg, ok := repo.configs[tenantA]
	require.True(t, ok)
	assert.Equal(t, "20-11111111-1", cfg.CUITEmsior)
	assert.Equal(t, "produccion", cfg.Modo)
	assert.Equal(t, tenantA, cfg.TenantID)
}

func TestConfigFiscal_TenantIsolation_ACannotSeeBConfig(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()
	tenantB := uuid.New()

	// Seed configs for both tenants
	repo.configs[tenantA] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantA,
		CUITEmsior:      "20-AAAA-A",
		RazonSocial:     "Kiosco A",
		CondicionFiscal: "Responsable Inscripto",
		PuntoDeVenta:    1,
		Modo:            "homologacion",
	}
	repo.configs[tenantB] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantB,
		CUITEmsior:      "20-BBBB-B",
		RazonSocial:     "Kiosco B",
		CondicionFiscal: "Monotributista",
		PuntoDeVenta:    3,
		Modo:            "produccion",
	}

	// Tenant A should only see their config
	ctxA := ctxWithTenant(tenantA)
	respA, err := svc.ObtenerConfiguracion(ctxA)
	require.NoError(t, err)
	assert.Equal(t, "20-AAAA-A", respA.CUITEmsior)
	assert.Equal(t, "Kiosco A", respA.RazonSocial)

	// Tenant B should only see their config
	ctxB := ctxWithTenant(tenantB)
	respB, err := svc.ObtenerConfiguracion(ctxB)
	require.NoError(t, err)
	assert.Equal(t, "20-BBBB-B", respB.CUITEmsior)
	assert.Equal(t, "Kiosco B", respB.RazonSocial)
}

func TestConfigFiscal_TenantIsolation_UpdateDoesNotAffectOther(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()
	tenantB := uuid.New()

	// Seed config for tenant B
	repo.configs[tenantB] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantB,
		CUITEmsior:      "20-BBBB-B",
		RazonSocial:     "Kiosco B",
		CondicionFiscal: "Monotributista",
		PuntoDeVenta:    3,
		Modo:            "produccion",
	}

	// Tenant A updates their config
	ctxA := ctxWithTenant(tenantA)
	err := svc.ActualizarConfiguracion(ctxA, dto.ConfiguracionFiscalRequest{
		CUITEmsior:      "20-AAAA-A",
		RazonSocial:     "Kiosco A Nuevo",
		CondicionFiscal: "Responsable Inscripto",
		PuntoDeVenta:    5,
		Modo:            "homologacion",
	})
	require.NoError(t, err)

	// Tenant B's config should be untouched
	cfgB := repo.configs[tenantB]
	assert.Equal(t, "20-BBBB-B", cfgB.CUITEmsior)
	assert.Equal(t, "Kiosco B", cfgB.RazonSocial)
	assert.Equal(t, 3, cfgB.PuntoDeVenta)
}

func TestConfigFiscal_ActualizarConfiguracion_PreservesExistingCerts(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()

	crt := "existing-crt-base64"
	key := "existing-key-base64"
	repo.configs[tenantA] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantA,
		CUITEmsior:      "20-11111111-1",
		RazonSocial:     "Old Name",
		CondicionFiscal: "Monotributista",
		PuntoDeVenta:    1,
		Modo:            "homologacion",
		CertificadoCrt:  &crt,
		CertificadoKey:  &key,
	}

	ctx := ctxWithTenant(tenantA)
	// Update without sending new certs
	err := svc.ActualizarConfiguracion(ctx, dto.ConfiguracionFiscalRequest{
		CUITEmsior:      "20-11111111-1",
		RazonSocial:     "New Name",
		CondicionFiscal: "Monotributista",
		PuntoDeVenta:    1,
		Modo:            "produccion",
	})
	require.NoError(t, err)

	cfg := repo.configs[tenantA]
	assert.Equal(t, "New Name", cfg.RazonSocial)
	assert.Equal(t, "produccion", cfg.Modo)
	require.NotNil(t, cfg.CertificadoCrt)
	assert.Equal(t, "existing-crt-base64", *cfg.CertificadoCrt)
	require.NotNil(t, cfg.CertificadoKey)
	assert.Equal(t, "existing-key-base64", *cfg.CertificadoKey)
}

func TestConfigFiscal_ObtenerConfiguracion_NeverExposesCertContent(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()

	crt := "super-secret-cert"
	key := "super-secret-key"
	repo.configs[tenantA] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantA,
		CUITEmsior:      "20-12345678-9",
		RazonSocial:     "Test",
		CondicionFiscal: "RI",
		PuntoDeVenta:    1,
		Modo:            "homologacion",
		CertificadoCrt:  &crt,
		CertificadoKey:  &key,
	}

	ctx := ctxWithTenant(tenantA)
	resp, err := svc.ObtenerConfiguracion(ctx)
	require.NoError(t, err)

	// The DTO should only tell if certs exist, not expose their content
	assert.True(t, resp.TieneCertificadoCrt)
	assert.True(t, resp.TieneCertificadoKey)

	// Marshal to JSON and check no cert content leaked
	jsonBytes, _ := json.Marshal(resp)
	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, "super-secret-cert")
	assert.NotContains(t, jsonStr, "super-secret-key")
}

func TestConfigFiscal_NoTenantInContext_ReturnsError(t *testing.T) {
	svc, _ := newConfigFiscalTestService()

	// No tenant_id in context
	_, err := svc.ObtenerConfiguracion(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant_id not found")
}

// ── Tests: Handler Layer ────────────────────────────────────────────────────

func TestConfigFiscalHandler_GET_ReturnsTenantConfig(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()

	repo.configs[tenantA] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantA,
		CUITEmsior:      "20-99999999-9",
		RazonSocial:     "Handler Test",
		CondicionFiscal: "RI",
		PuntoDeVenta:    4,
		Modo:            "produccion",
	}

	router := setupConfigFiscalRouter(svc)
	w := doGetConfigFiscal(t, router, tenantA)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	var data dto.ConfiguracionFiscalResponse
	err = json.Unmarshal(body["data"], &data)
	require.NoError(t, err)

	assert.Equal(t, "20-99999999-9", data.CUITEmsior)
	assert.Equal(t, "Handler Test", data.RazonSocial)
	assert.Equal(t, 4, data.PuntoDeVenta)
	assert.Equal(t, "produccion", data.Modo)
}

func TestConfigFiscalHandler_GET_TenantIsolation(t *testing.T) {
	svc, repo := newConfigFiscalTestService()
	tenantA := uuid.New()
	tenantB := uuid.New()

	repo.configs[tenantA] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantA,
		CUITEmsior:      "20-AAAA-A",
		RazonSocial:     "Tenant A",
		CondicionFiscal: "RI",
		PuntoDeVenta:    1,
		Modo:            "homologacion",
	}
	repo.configs[tenantB] = &model.ConfiguracionFiscal{
		ID:              uuid.New(),
		TenantID:        tenantB,
		CUITEmsior:      "20-BBBB-B",
		RazonSocial:     "Tenant B",
		CondicionFiscal: "Monotributista",
		PuntoDeVenta:    2,
		Modo:            "produccion",
	}

	router := setupConfigFiscalRouter(svc)

	// Tenant A sees only their config
	wA := doGetConfigFiscal(t, router, tenantA)
	assert.Equal(t, http.StatusOK, wA.Code)
	var bodyA map[string]json.RawMessage
	json.Unmarshal(wA.Body.Bytes(), &bodyA)
	var dataA dto.ConfiguracionFiscalResponse
	json.Unmarshal(bodyA["data"], &dataA)
	assert.Equal(t, "20-AAAA-A", dataA.CUITEmsior)

	// Tenant B sees only their config
	wB := doGetConfigFiscal(t, router, tenantB)
	assert.Equal(t, http.StatusOK, wB.Code)
	var bodyB map[string]json.RawMessage
	json.Unmarshal(wB.Body.Bytes(), &bodyB)
	var dataB dto.ConfiguracionFiscalResponse
	json.Unmarshal(bodyB["data"], &dataB)
	assert.Equal(t, "20-BBBB-B", dataB.CUITEmsior)
}

func TestConfigFiscalHandler_PUT_ValidationRejectsEmptyFields(t *testing.T) {
	svc, _ := newConfigFiscalTestService()
	tenantA := uuid.New()

	router := setupConfigFiscalRouter(svc)

	// PUT with empty multipart form (missing required fields)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/v1/configuracion/fiscal", nil)
	req.Header.Set("X-Test-Tenant-ID", tenantA.String())
	req.Header.Set("Content-Type", "multipart/form-data; boundary=testboundary")
	// Empty body — will fail ParseMultipartForm
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConfigFiscalHandler_GET_NoTenant_Returns500(t *testing.T) {
	svc, _ := newConfigFiscalTestService()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewConfiguracionFiscalHandler(svc)
	// No tenant middleware — context has no tenant_id
	r.GET("/v1/configuracion/fiscal", h.Obtener)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/v1/configuracion/fiscal", nil)
	r.ServeHTTP(w, req)

	// Service returns error because no tenant_id in context
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
