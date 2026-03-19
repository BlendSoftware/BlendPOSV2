package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"blendpos/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeTenantWithFeatures(features map[string]bool) *model.Tenant {
	planID := uuid.New()
	featuresJSON, _ := json.Marshal(features)
	return &model.Tenant{
		ID:     uuid.New(),
		Nombre: "Test Tenant",
		PlanID: &planID,
		Plan: &model.Plan{
			ID:       planID,
			Nombre:   "Pro",
			Features: featuresJSON,
		},
	}
}

// ---------------------------------------------------------------------------
// Tests: RequireFeature
// ---------------------------------------------------------------------------

func TestRequireFeature_AllowsWhenFeatureEnabled(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: makeTenantWithFeatures(map[string]bool{
			"analytics_avanzados": true,
			"export_csv":          true,
		}),
	}

	r := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireFeature_BlocksWhenFeatureDisabled(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: makeTenantWithFeatures(map[string]bool{
			"analytics_avanzados": false,
			"export_csv":          true,
		}),
	}

	r := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body FeatureFlagError
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "analytics_avanzados", body.Feature)
	assert.Equal(t, "/planes", body.UpgradeURL)
}

func TestRequireFeature_BlocksWhenFeatureMissing(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: makeTenantWithFeatures(map[string]bool{
			"export_csv": true,
		}),
	}

	r := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireFeature_FailOpenWhenNoPlan(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenant: &model.Tenant{ID: tid, Nombre: "No Plan", Plan: nil},
	}

	r := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should fail-open when tenant has no plan")
}

func TestRequireFeature_FailOpenOnDBError(t *testing.T) {
	tid := uuid.New()
	repo := &stubTenantRepo{
		tenantErr: assert.AnError,
	}

	r := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "should fail-open when DB is unreachable")
}

func TestRequireFeature_FailOpenOnEmptyFeatures(t *testing.T) {
	tid := uuid.New()
	planID := uuid.New()
	repo := &stubTenantRepo{
		tenant: &model.Tenant{
			ID:     tid,
			Nombre: "Empty Features",
			PlanID: &planID,
			Plan: &model.Plan{
				ID:       planID,
				Nombre:   "Kiosco",
				Features: json.RawMessage(`{}`),
			},
		},
	}

	r := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r.ServeHTTP(w, req)

	// Empty features = feature not present = blocked
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: parseFeatures
// ---------------------------------------------------------------------------

func TestParseFeatures_ValidJSON(t *testing.T) {
	raw := json.RawMessage(`{"analytics_avanzados": true, "export_csv": false}`)
	features, err := parseFeatures(raw)
	require.NoError(t, err)
	assert.True(t, features["analytics_avanzados"])
	assert.False(t, features["export_csv"])
}

func TestParseFeatures_EmptyJSON(t *testing.T) {
	for _, input := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`{}`), json.RawMessage(``)} {
		features, err := parseFeatures(input)
		require.NoError(t, err)
		assert.NotNil(t, features)
		assert.Empty(t, features)
	}
}

func TestParseFeatures_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{not valid json}`)
	_, err := parseFeatures(raw)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Tests: FeatureFlagError JSON structure
// ---------------------------------------------------------------------------

func TestFeatureFlagError_JSONStructure(t *testing.T) {
	ffe := FeatureFlagError{
		Error:      "Función no disponible en tu plan actual",
		Feature:    "analytics_avanzados",
		UpgradeURL: "/planes",
	}

	data, err := json.Marshal(ffe)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "analytics_avanzados", m["feature"])
	assert.Equal(t, "/planes", m["upgrade_url"])
}

// ---------------------------------------------------------------------------
// Tests: Multiple features — check isolation
// ---------------------------------------------------------------------------

func TestRequireFeature_DifferentFeatures_Independent(t *testing.T) {
	tid := uuid.New()
	features := map[string]bool{
		"analytics_avanzados": false,
		"export_csv":          true,
		"multi_terminal":      true,
	}
	repo := &stubTenantRepo{tenant: makeTenantWithFeatures(features)}

	// analytics_avanzados should be blocked
	r1 := setupPlanTestRouter(RequireFeature("analytics_avanzados", repo, nil), tid)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r1.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusForbidden, w1.Code)

	// export_csv should be allowed
	r2 := setupPlanTestRouter(RequireFeature("export_csv", repo, nil), tid)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/test", nil)
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}
