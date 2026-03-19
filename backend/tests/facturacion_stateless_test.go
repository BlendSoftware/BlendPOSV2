package tests

// facturacion_stateless_test.go
// F1-4: Tests for stateless multi-CUIT AFIP sidecar integration.
// Verifies:
//   - Base64 -> PEM decoding in buildAFIPPayload
//   - Cert passthrough when already PEM
//   - Worker builds correct payload with certs from ConfiguracionFiscal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blendpos/internal/dto"
	"blendpos/internal/infra"
	"blendpos/internal/model"
	"blendpos/internal/worker"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleCertPEM = `-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQD2Bp1fMN7GnTANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC7
-----END CERTIFICATE-----`

const sampleKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAu6OKp3utEwd6T2GYDxJ58UIp9pzWkzgTDNG14SugXTH5dDGz
v95rsR+UqW52xI29C5Sj6ID5AQvCV3R8y2pgTX8LrAWtSTr9kWIhtIx5l8RxPKvy
-----END RSA PRIVATE KEY-----`

// -- Stub that captures the AFIP payload sent to the sidecar -------------------

type capturingAFIPClient struct {
	lastPayload *infra.AFIPPayload
	response    *infra.AFIPResponse
	err         error
}

func (c *capturingAFIPClient) Facturar(_ context.Context, p infra.AFIPPayload) (*infra.AFIPResponse, error) {
	c.lastPayload = &p
	return c.response, c.err
}
func (c *capturingAFIPClient) GetSidecarURL() string   { return "http://fake-sidecar:8001" }
func (c *capturingAFIPClient) GetInternalToken() string { return "test-token" }

var _ infra.AFIPClient = (*capturingAFIPClient)(nil)

// -- Stub ConfiguracionFiscalProvider with base64-encoded certs ----------------

type stubConfigFiscalProvider struct {
	config *model.ConfiguracionFiscal
}

func (s *stubConfigFiscalProvider) ObtenerConfiguracion(_ context.Context) (*dto.ConfiguracionFiscalResponse, error) {
	return nil, nil
}
func (s *stubConfigFiscalProvider) ObtenerConfiguracionCompleta(_ context.Context) (*model.ConfiguracionFiscal, error) {
	if s.config == nil {
		return nil, fmt.Errorf("no config")
	}
	return s.config, nil
}

var _ worker.ConfiguracionFiscalProvider = (*stubConfigFiscalProvider)(nil)

// -- Tests: Base64 decode in payload building ---------------------------------

func TestBuildAFIPPayload_DecodesBase64CertsFromDB(t *testing.T) {
	// Simulate what the handler does: base64-encode the PEM content.
	// This is what gets stored in ConfiguracionFiscal.CertificadoCrt/Key.
	crtBase64 := base64.StdEncoding.EncodeToString([]byte(sampleCertPEM))
	keyBase64 := base64.StdEncoding.EncodeToString([]byte(sampleKeyPEM))

	afipClient := &capturingAFIPClient{
		response: &infra.AFIPResponse{
			Resultado:         "A",
			CAE:               "12345678901234",
			CAEVencimiento:    "20260401",
			NumeroComprobante: 1,
			PuntoDeVenta:      1,
		},
	}

	comprobanteRepo := newStubComprobanteRepo()
	ventaRepo := newStubVentaRepoFacturacion()
	tmpDir := t.TempDir()

	venta := buildVentaConItems()
	ventaRepo.ventas[venta.ID] = venta

	configProvider := &stubConfigFiscalProvider{
		config: &model.ConfiguracionFiscal{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			CUITEmsior:      "20123456789",
			RazonSocial:     "Test Kiosco",
			CondicionFiscal: "Monotributo",
			PuntoDeVenta:    1,
			CertificadoCrt:  &crtBase64,
			CertificadoKey:  &keyBase64,
			Modo:            "homologacion",
		},
	}

	cb := infra.NewCircuitBreaker(infra.DefaultCBConfig())
	w := worker.NewFacturacionWorker(afipClient, cb, comprobanteRepo, ventaRepo, nil, tmpDir, configProvider)

	payload := worker.FacturacionJobPayload{
		VentaID:         venta.ID.String(),
		TipoComprobante: "factura_c",
	}
	w.Process(context.Background(), mustJSON(payload))

	// Verify the AFIP payload was sent with DECODED PEM, not raw base64
	require.NotNil(t, afipClient.lastPayload, "AFIP client should have been called")
	assert.True(t, strings.HasPrefix(afipClient.lastPayload.CertPEM, "-----BEGIN CERTIFICATE-----"),
		"CertPEM should be decoded PEM, got: %s", truncate(afipClient.lastPayload.CertPEM, 50))
	assert.True(t, strings.HasPrefix(afipClient.lastPayload.KeyPEM, "-----BEGIN RSA PRIVATE KEY-----"),
		"KeyPEM should be decoded PEM, got: %s", truncate(afipClient.lastPayload.KeyPEM, 50))
	assert.Equal(t, "homologacion", afipClient.lastPayload.Modo)
	assert.Equal(t, "20123456789", afipClient.lastPayload.CUITEmisor)
}

func TestBuildAFIPPayload_PassthroughWhenAlreadyPEM(t *testing.T) {
	// Edge case: if someone stored PEM directly (no base64), it should pass through.
	certPEM := sampleCertPEM
	keyPEM := sampleKeyPEM

	afipClient := &capturingAFIPClient{
		response: &infra.AFIPResponse{
			Resultado:         "A",
			CAE:               "12345678901234",
			CAEVencimiento:    "20260401",
			NumeroComprobante: 1,
			PuntoDeVenta:      1,
		},
	}

	comprobanteRepo := newStubComprobanteRepo()
	ventaRepo := newStubVentaRepoFacturacion()
	tmpDir := t.TempDir()

	venta := buildVentaConItems()
	ventaRepo.ventas[venta.ID] = venta

	configProvider := &stubConfigFiscalProvider{
		config: &model.ConfiguracionFiscal{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			CUITEmsior:      "20987654321",
			RazonSocial:     "Test Kiosco 2",
			CondicionFiscal: "Monotributo",
			PuntoDeVenta:    2,
			CertificadoCrt:  &certPEM,
			CertificadoKey:  &keyPEM,
			Modo:            "produccion",
		},
	}

	cb := infra.NewCircuitBreaker(infra.DefaultCBConfig())
	w := worker.NewFacturacionWorker(afipClient, cb, comprobanteRepo, ventaRepo, nil, tmpDir, configProvider)

	payload := worker.FacturacionJobPayload{
		VentaID:         venta.ID.String(),
		TipoComprobante: "factura_c",
	}
	w.Process(context.Background(), mustJSON(payload))

	require.NotNil(t, afipClient.lastPayload)
	assert.Equal(t, sampleCertPEM, afipClient.lastPayload.CertPEM)
	assert.Equal(t, sampleKeyPEM, afipClient.lastPayload.KeyPEM)
	assert.Equal(t, "produccion", afipClient.lastPayload.Modo)
}

func TestBuildAFIPPayload_NoCertsWhenConfigMissing(t *testing.T) {
	// When ConfiguracionFiscal has no certs, CertPEM/KeyPEM should be empty.
	afipClient := &capturingAFIPClient{
		response: &infra.AFIPResponse{
			Resultado:         "A",
			CAE:               "12345678901234",
			CAEVencimiento:    "20260401",
			NumeroComprobante: 1,
			PuntoDeVenta:      1,
		},
	}

	comprobanteRepo := newStubComprobanteRepo()
	ventaRepo := newStubVentaRepoFacturacion()
	tmpDir := t.TempDir()

	venta := buildVentaConItems()
	ventaRepo.ventas[venta.ID] = venta

	configProvider := &stubConfigFiscalProvider{
		config: &model.ConfiguracionFiscal{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			CUITEmsior:      "20555555555",
			RazonSocial:     "No Certs Kiosco",
			CondicionFiscal: "Monotributo",
			PuntoDeVenta:    1,
			CertificadoCrt:  nil,
			CertificadoKey:  nil,
			Modo:            "homologacion",
		},
	}

	cb := infra.NewCircuitBreaker(infra.DefaultCBConfig())
	w := worker.NewFacturacionWorker(afipClient, cb, comprobanteRepo, ventaRepo, nil, tmpDir, configProvider)

	payload := worker.FacturacionJobPayload{
		VentaID:         venta.ID.String(),
		TipoComprobante: "factura_c",
	}
	w.Process(context.Background(), mustJSON(payload))

	require.NotNil(t, afipClient.lastPayload)
	assert.Empty(t, afipClient.lastPayload.CertPEM, "CertPEM should be empty when no cert in config")
	assert.Empty(t, afipClient.lastPayload.KeyPEM, "KeyPEM should be empty when no key in config")
}

// TestSidecarReceivesPEMViaCertFields validates the end-to-end flow:
// Go backend sends decoded PEM -> sidecar receives it correctly.
func TestSidecarReceivesPEMViaCertFields(t *testing.T) {
	// Spin up a fake sidecar that captures the request body
	var receivedPayload map[string]interface{}
	fakeSidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"resultado": "A",
			"numero_comprobante": 1,
			"fecha_comprobante": "20260316",
			"cae": "99887766554433",
			"cae_vencimiento": "20260326"
		}`))
	}))
	defer fakeSidecar.Close()

	// Build a real AFIPClient pointing at the fake sidecar
	afipClient := infra.NewAFIPClient(fakeSidecar.URL, "test-token")

	// Send a payload with PEM certs
	payload := infra.AFIPPayload{
		CUITEmisor:       "20123456789",
		PuntoDeVenta:     1,
		TipoComprobante:  11,
		TipoDocReceptor:  99,
		NroDocReceptor:   "0",
		Concepto:         1,
		ImporteNeto:      "1000.00",
		ImporteExento:    "0.00",
		ImporteIVA:       "0.00",
		ImporteTributos:  "0.00",
		ImporteTotal:     "1000.00",
		Moneda:           "PES",
		CotizacionMoneda: 1.0,
		VentaID:          uuid.New().String(),
		CertPEM:          sampleCertPEM,
		KeyPEM:           sampleKeyPEM,
		Modo:             "homologacion",
	}

	resp, err := afipClient.Facturar(context.Background(), payload)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "A", resp.Resultado)
	assert.Equal(t, "99887766554433", resp.CAE)

	// Verify the fake sidecar received PEM content, not base64
	certPEM, _ := receivedPayload["cert_pem"].(string)
	keyPEM, _ := receivedPayload["key_pem"].(string)
	assert.True(t, strings.HasPrefix(certPEM, "-----BEGIN CERTIFICATE-----"),
		"Sidecar should receive PEM cert content")
	assert.True(t, strings.HasPrefix(keyPEM, "-----BEGIN RSA PRIVATE KEY-----"),
		"Sidecar should receive PEM key content")
}

// -- helpers ------------------------------------------------------------------

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Ensure decimal import is used (needed by stubs in other test files)
var _ = decimal.Zero
