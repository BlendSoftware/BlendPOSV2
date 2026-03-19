package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blendpos/internal/dto"
	"blendpos/internal/handler"
	"blendpos/internal/model"
	"blendpos/internal/service"
	"blendpos/internal/tenantctx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ── Mock MPClient ─────────────────────────────────────────────────────────────

type mockMPClient struct {
	createFunc func(ctx context.Context, req service.MPCreateSubscriptionRequest) (*service.MPCreateSubscriptionResponse, error)
}

func (m *mockMPClient) CreateSubscription(ctx context.Context, req service.MPCreateSubscriptionRequest) (*service.MPCreateSubscriptionResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return &service.MPCreateSubscriptionResponse{
		SubscriptionID: "MP-SUB-123",
		CheckoutURL:    "https://www.mercadopago.com.ar/checkout/v1/redirect?pref_id=abc123",
	}, nil
}

// ── Stub Subscription Repository ──────────────────────────────────────────────

type stubSubscriptionRepo struct {
	subs map[string]*model.Subscription // keyed by ID string
}

func newStubSubscriptionRepo() *stubSubscriptionRepo {
	return &stubSubscriptionRepo{
		subs: make(map[string]*model.Subscription),
	}
}

func (r *stubSubscriptionRepo) Create(_ context.Context, s *model.Subscription) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	r.subs[s.ID.String()] = s
	return nil
}

func (r *stubSubscriptionRepo) FindByID(_ context.Context, id uuid.UUID) (*model.Subscription, error) {
	s, ok := r.subs[id.String()]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return s, nil
}

func (r *stubSubscriptionRepo) FindActiveByTenantID(_ context.Context, tenantID uuid.UUID) (*model.Subscription, error) {
	for _, s := range r.subs {
		if s.TenantID == tenantID && (s.Status == "active" || s.Status == "pending") {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *stubSubscriptionRepo) FindByMPSubscriptionID(_ context.Context, mpSubID string) (*model.Subscription, error) {
	for _, s := range r.subs {
		if s.MPSubscriptionID != nil && *s.MPSubscriptionID == mpSubID {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *stubSubscriptionRepo) Update(_ context.Context, s *model.Subscription) error {
	s.UpdatedAt = time.Now()
	r.subs[s.ID.String()] = s
	return nil
}

// ── Test Helpers ──────────────────────────────────────────────────────────────

func newBillingTestDeps() (service.BillingService, *stubSubscriptionRepo, *stubTenantRepo, *mockMPClient) {
	subRepo := newStubSubscriptionRepo()
	tenantRepo := newStubTenantRepo()
	mpClient := &mockMPClient{}
	svc := service.NewBillingService(subRepo, tenantRepo, mpClient)
	return svc, subRepo, tenantRepo, mpClient
}

func setupTenantWithPlan(tenantRepo *stubTenantRepo) (uuid.UUID, uuid.UUID) {
	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	tenantID := uuid.New()
	tenant := &model.Tenant{
		ID:     tenantID,
		Slug:   "test-kiosco",
		Nombre: "Test Kiosco",
		PlanID: &proID,
		Activo: true,
	}
	tenantRepo.tenants[tenant.Slug] = tenant
	return tenantID, proID
}

func billingCtxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), tenantctx.Key, tenantID)
}

// ── Tests: Subscribe ──────────────────────────────────────────────────────────

func TestSubscribe_CreatesSubscriptionAndCallsMP(t *testing.T) {
	svc, subRepo, tenantRepo, mpClient := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	var mpCalled bool
	mpClient.createFunc = func(_ context.Context, req service.MPCreateSubscriptionRequest) (*service.MPCreateSubscriptionResponse, error) {
		mpCalled = true
		assert.Equal(t, tenantID.String(), req.TenantID)
		return &service.MPCreateSubscriptionResponse{
			SubscriptionID: "MP-SUB-456",
			CheckoutURL:    "https://mp.com/checkout/456",
		}, nil
	}

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	// Add Pro plan to tenantRepo
	tenantRepo.plans[proID.String()] = &model.Plan{
		ID:     proID,
		Nombre: "Pro",
		Activo: true,
	}

	ctx := billingCtxWithTenant(tenantID)
	resp, err := svc.Subscribe(ctx, dto.SubscribeRequest{PlanID: proID.String()})

	require.NoError(t, err)
	assert.True(t, mpCalled, "MPClient.CreateSubscription should have been called")
	assert.Equal(t, "https://mp.com/checkout/456", resp.CheckoutURL)
	assert.Equal(t, "pending", resp.Status)
	assert.NotEmpty(t, resp.SubscriptionID)

	// Verify subscription was saved
	assert.Len(t, subRepo.subs, 1)
	for _, sub := range subRepo.subs {
		assert.Equal(t, tenantID, sub.TenantID)
		assert.Equal(t, "pending", sub.Status)
		require.NotNil(t, sub.MPSubscriptionID)
		assert.Equal(t, "MP-SUB-456", *sub.MPSubscriptionID)
	}
}

func TestSubscribe_InvalidPlan_ReturnsError(t *testing.T) {
	svc, _, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	ctx := billingCtxWithTenant(tenantID)
	fakePlanID := uuid.New()
	_, err := svc.Subscribe(ctx, dto.SubscribeRequest{PlanID: fakePlanID.String()})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan no encontrado")
}

// ── Tests: Webhook ────────────────────────────────────────────────────────────

func TestWebhook_PaymentAuthorized_ActivatesSubscriptionAndUpgradesPlan(t *testing.T) {
	svc, subRepo, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	// Pre-create a pending subscription
	mpSubID := "MP-SUB-789"
	sub := &model.Subscription{
		TenantID:           tenantID,
		PlanID:             proID,
		MPSubscriptionID:   &mpSubID,
		Status:             "pending",
	}
	_ = subRepo.Create(context.Background(), sub)

	err := svc.HandleWebhook(context.Background(), service.WebhookEvent{
		Type:           "subscription_preapproval",
		Action:         "updated",
		SubscriptionID: mpSubID,
		Status:         "authorized",
	})

	require.NoError(t, err)

	// Verify subscription is now active
	updated, err := subRepo.FindByMPSubscriptionID(context.Background(), mpSubID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
	assert.NotNil(t, updated.CurrentPeriodStart)
	assert.NotNil(t, updated.CurrentPeriodEnd)

	// Verify tenant plan was upgraded
	tenant, _ := tenantRepo.FindTenantByID(context.Background(), tenantID)
	require.NotNil(t, tenant.PlanID)
	assert.Equal(t, proID, *tenant.PlanID)
}

func TestWebhook_PaymentPaused_PausesSubscription(t *testing.T) {
	svc, subRepo, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	mpSubID := "MP-SUB-PAUSE"
	sub := &model.Subscription{
		TenantID:         tenantID,
		PlanID:           proID,
		MPSubscriptionID: &mpSubID,
		Status:           "active",
	}
	_ = subRepo.Create(context.Background(), sub)

	err := svc.HandleWebhook(context.Background(), service.WebhookEvent{
		SubscriptionID: mpSubID,
		Status:         "paused",
	})

	require.NoError(t, err)

	updated, _ := subRepo.FindByMPSubscriptionID(context.Background(), mpSubID)
	assert.Equal(t, "paused", updated.Status)
}

func TestWebhook_UnknownSubscription_IgnoredGracefully(t *testing.T) {
	svc, _, _, _ := newBillingTestDeps()

	err := svc.HandleWebhook(context.Background(), service.WebhookEvent{
		SubscriptionID: "MP-UNKNOWN-999",
		Status:         "authorized",
	})

	// Should not error — idempotent handling
	assert.NoError(t, err)
}

// ── Tests: GetStatus ──────────────────────────────────────────────────────────

func TestGetStatus_ActiveSubscription(t *testing.T) {
	svc, subRepo, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	proPlan := &model.Plan{ID: proID, Nombre: "Pro", Activo: true}
	tenantRepo.plans[proID.String()] = proPlan

	now := time.Now()
	end := now.AddDate(0, 1, 0)
	sub := &model.Subscription{
		TenantID:           tenantID,
		PlanID:             proID,
		Status:             "active",
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &end,
		Plan:               proPlan,
	}
	_ = subRepo.Create(context.Background(), sub)

	ctx := billingCtxWithTenant(tenantID)
	resp, err := svc.GetStatus(ctx)

	require.NoError(t, err)
	assert.True(t, resp.HasSubscription)
	assert.Equal(t, "active", resp.Status)
	assert.Equal(t, "Pro", resp.PlanNombre)
	assert.NotNil(t, resp.PeriodEnd)
}

func TestGetStatus_NoSubscription(t *testing.T) {
	svc, _, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	ctx := billingCtxWithTenant(tenantID)
	resp, err := svc.GetStatus(ctx)

	require.NoError(t, err)
	assert.False(t, resp.HasSubscription)
	assert.Empty(t, resp.Status)
}

// ── Tests: Handler layer (HTTP) ───────────────────────────────────────────────

func TestSubscribeHandler_Success(t *testing.T) {
	svc, _, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	tenantRepo.plans[proID.String()] = &model.Plan{ID: proID, Nombre: "Pro", Activo: true}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewBillingHandler(svc)
	r.POST("/v1/billing/subscribe", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, h.Subscribe)

	body, _ := json.Marshal(dto.SubscribeRequest{PlanID: proID.String()})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/billing/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.SubscribeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.SubscriptionID)
	assert.NotEmpty(t, resp.CheckoutURL)
	assert.Equal(t, "pending", resp.Status)
}

func TestWebhookHandler_ProcessesEvent(t *testing.T) {
	svc, subRepo, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	mpSubID := "MP-HANDLER-TEST"
	sub := &model.Subscription{
		TenantID:         tenantID,
		PlanID:           proID,
		MPSubscriptionID: &mpSubID,
		Status:           "pending",
	}
	_ = subRepo.Create(context.Background(), sub)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewBillingHandler(svc)
	r.POST("/v1/billing/webhook", h.Webhook)

	payload := map[string]interface{}{
		"type":   "subscription_preapproval",
		"action": "updated",
		"data": map[string]interface{}{
			"id":     mpSubID,
			"status": "authorized",
		},
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/billing/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify subscription was activated
	updated, _ := subRepo.FindByMPSubscriptionID(context.Background(), mpSubID)
	assert.Equal(t, "active", updated.Status)
}

func TestWebhookHandler_MissingSignature_StillProcesses(t *testing.T) {
	// For now (TODO), we allow webhooks without signature.
	// When MP signature verification is implemented, this test should
	// assert 401 instead.
	svc, _, _, _ := newBillingTestDeps()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewBillingHandler(svc)
	r.POST("/v1/billing/webhook", h.Webhook)

	payload := map[string]interface{}{
		"type":            "payment",
		"subscription_id": "MP-NONEXISTENT",
		"status":          "authorized",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/billing/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Signature header
	r.ServeHTTP(w, req)

	// Should still return 200 (unknown subscription is ignored gracefully)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetStatusHandler_Success(t *testing.T) {
	svc, _, tenantRepo, _ := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewBillingHandler(svc)
	r.GET("/v1/billing/status", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), tenantctx.Key, tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, h.GetStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/v1/billing/status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.BillingStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.HasSubscription)
}

func TestSubscribe_MPClientError_ReturnsError(t *testing.T) {
	svc, _, tenantRepo, mpClient := newBillingTestDeps()
	tenantID, _ := setupTenantWithPlan(tenantRepo)

	proID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	tenantRepo.plans[proID.String()] = &model.Plan{ID: proID, Nombre: "Pro", Activo: true}

	mpClient.createFunc = func(_ context.Context, _ service.MPCreateSubscriptionRequest) (*service.MPCreateSubscriptionResponse, error) {
		return nil, errors.New("MP API timeout")
	}

	ctx := billingCtxWithTenant(tenantID)
	_, err := svc.Subscribe(ctx, dto.SubscribeRequest{PlanID: proID.String()})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MercadoPago")
}
