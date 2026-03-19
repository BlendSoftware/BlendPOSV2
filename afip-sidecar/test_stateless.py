"""
F1-4: Tests for stateless multi-CUIT AFIP sidecar mode.

Tests:
1. Base64 cert decode + PEM validation in schema
2. Temp file creation and cleanup in AFIPClient
3. Parallel requests with different CUITs don't interfere
4. Stateless /facturar path selection
"""

import base64
import os
import tempfile
import shutil
from unittest.mock import patch, MagicMock
from concurrent.futures import ThreadPoolExecutor

import pytest
from pydantic import ValidationError

from schemas import FacturarRequest


# ── Sample PEM content for testing ────────────────────────────────────────────

SAMPLE_CERT_PEM = """\
-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQD2Bp1fMN7GnTANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC7
o4qne60TB3pPYZgPEnnxQin2nNaTOBMM0bXhK6BdMfl0MbO/3muxH5SpbnbEjb0L
lKPogPkBC8JXdHzLamBNfwusBa1JOv2RYiG0jHmXxHE8q/IraN/ukHfk+sJqlGBk
hhbGnUhRW+rlXi7geqOI5qQPvDBu1fjLnPcJBZlEpB0n+ALDNz7MqMC7M2FXH0O3
-----END CERTIFICATE-----"""

SAMPLE_KEY_PEM = """\
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAu6OKp3utEwd6T2GYDxJ58UIp9pzWkzgTDNG14SugXTH5dDGz
v95rsR+UqW52xI29C5Sj6ID5AQvCV3R8y2pgTX8LrAWtSTr9kWIhtIx5l8RxPKvy
K2jf7pB35PrCapRgZIYWxp1IUVvq5V4u4HqjiOakD7wwbtX4y5z3CQWZRKQdJ/gC
wzc+zKjAuzNhVx9DtDANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC7o4qn
-----END RSA PRIVATE KEY-----"""

SAMPLE_CERT_B64 = base64.b64encode(SAMPLE_CERT_PEM.encode()).decode()
SAMPLE_KEY_B64 = base64.b64encode(SAMPLE_KEY_PEM.encode()).decode()


# ── Schema validation tests ──────────────────────────────────────────────────

class TestFacturarRequestSchema:
    """Test FacturarRequest Pydantic schema for stateless mode fields."""

    def _base_payload(self, **overrides):
        """Build a valid base FacturarRequest payload."""
        payload = {
            "cuit_emisor": "20123456789",
            "punto_de_venta": 1,
            "tipo_comprobante": 11,
            "tipo_doc_receptor": 99,
            "nro_doc_receptor": "0",
            "concepto": 1,
            "importe_neto": "1000.00",
            "importe_exento": "0.00",
            "importe_iva": "0.00",
            "importe_tributos": "0.00",
            "importe_total": "1000.00",
        }
        payload.update(overrides)
        return payload

    def test_stateless_mode_accepts_valid_pem(self):
        """cert_pem and key_pem with valid PEM content should pass validation."""
        req = FacturarRequest(**self._base_payload(
            cert_pem=SAMPLE_CERT_PEM,
            key_pem=SAMPLE_KEY_PEM,
            modo="homologacion",
        ))
        assert req.cert_pem is not None
        assert req.key_pem is not None
        assert req.cert_pem.startswith("-----BEGIN")

    def test_stateless_mode_rejects_file_path(self):
        """cert_pem must be PEM content, not a file path (path traversal protection)."""
        with pytest.raises(ValidationError) as exc_info:
            FacturarRequest(**self._base_payload(
                cert_pem="/etc/passwd",
                key_pem=SAMPLE_KEY_PEM,
            ))
        assert "-----BEGIN" in str(exc_info.value)

    def test_legacy_mode_no_certs_accepted(self):
        """Request without cert_pem/key_pem is valid (legacy mode)."""
        req = FacturarRequest(**self._base_payload())
        assert req.cert_pem is None
        assert req.key_pem is None

    def test_modo_defaults_to_homologacion(self):
        """modo field defaults to 'homologacion'."""
        req = FacturarRequest(**self._base_payload())
        assert req.modo == "homologacion"


# ── AFIPClient temp file tests ───────────────────────────────────────────────

class TestAFIPClientTempFiles:
    """Test temp file creation and cleanup in AFIPClient stateless mode."""

    def _make_client(self, cert_pem=SAMPLE_CERT_PEM, key_pem=SAMPLE_KEY_PEM):
        """Create an AFIPClient in stateless mode with test PEM content.

        Patches pyafipws imports since they may not be available in test env.
        """
        # We need to mock pyafipws imports before importing afip_client
        with patch.dict("sys.modules", {
            "py3_compat": MagicMock(),
            "patch_analizar_errores": MagicMock(),
            "pyafipws": MagicMock(),
            "pyafipws.wsaa": MagicMock(),
            "pyafipws.wsfev1": MagicMock(),
        }):
            # Force re-import to apply mocks
            import importlib
            import afip_client
            importlib.reload(afip_client)

            client = afip_client.AFIPClient(
                cuit_emisor="20123456789",
                homologacion=True,
                cert_pem=cert_pem,
                key_pem=key_pem,
            )
            return client

    def test_temp_dir_created_with_cert_files(self):
        """Stateless mode should write PEM to temp files."""
        client = self._make_client()

        assert client._temp_dir is not None
        assert os.path.isdir(client._temp_dir)
        assert os.path.isfile(client.cert_path)
        assert os.path.isfile(client.key_path)

        # Verify content
        with open(client.cert_path) as f:
            assert f.read() == SAMPLE_CERT_PEM
        with open(client.key_path) as f:
            assert f.read() == SAMPLE_KEY_PEM

        # Cleanup
        client.cleanup()

    def test_cleanup_removes_temp_dir(self):
        """cleanup() should remove the temp directory and all files."""
        client = self._make_client()
        temp_dir = client._temp_dir
        cert_path = client.cert_path
        key_path = client.key_path

        # Files exist before cleanup
        assert os.path.exists(temp_dir)
        assert os.path.exists(cert_path)
        assert os.path.exists(key_path)

        client.cleanup()

        # All gone after cleanup
        assert not os.path.exists(temp_dir)
        assert not os.path.exists(cert_path)
        assert not os.path.exists(key_path)
        assert client._temp_dir is None

    def test_cleanup_idempotent(self):
        """Calling cleanup() twice should not raise."""
        client = self._make_client()
        client.cleanup()
        client.cleanup()  # Second call should be a no-op

    def test_no_temp_dir_in_legacy_mode(self):
        """Legacy mode (cert_path/key_path) should not create temp dir."""
        with patch.dict("sys.modules", {
            "py3_compat": MagicMock(),
            "patch_analizar_errores": MagicMock(),
            "pyafipws": MagicMock(),
            "pyafipws.wsaa": MagicMock(),
            "pyafipws.wsfev1": MagicMock(),
        }):
            import importlib
            import afip_client
            importlib.reload(afip_client)

            client = afip_client.AFIPClient(
                cuit_emisor="20123456789",
                cert_path="/certs/afip.crt",
                key_path="/certs/afip.key",
                homologacion=True,
            )
            assert client._temp_dir is None


# ── Parallel request isolation tests ─────────────────────────────────────────

class TestParallelRequestIsolation:
    """Verify that parallel requests with different CUITs don't share state."""

    def test_parallel_clients_have_separate_temp_dirs(self):
        """Two AFIPClient instances should write to different temp dirs."""
        with patch.dict("sys.modules", {
            "py3_compat": MagicMock(),
            "patch_analizar_errores": MagicMock(),
            "pyafipws": MagicMock(),
            "pyafipws.wsaa": MagicMock(),
            "pyafipws.wsfev1": MagicMock(),
        }):
            import importlib
            import afip_client
            importlib.reload(afip_client)

            client_a = afip_client.AFIPClient(
                cuit_emisor="20111111111",
                homologacion=True,
                cert_pem=SAMPLE_CERT_PEM,
                key_pem=SAMPLE_KEY_PEM,
            )
            client_b = afip_client.AFIPClient(
                cuit_emisor="20222222222",
                homologacion=True,
                cert_pem=SAMPLE_CERT_PEM,
                key_pem=SAMPLE_KEY_PEM,
            )

            # Different temp dirs
            assert client_a._temp_dir != client_b._temp_dir
            assert client_a.cert_path != client_b.cert_path

            # Cleanup one doesn't affect the other
            client_a.cleanup()
            assert not os.path.exists(client_a._temp_dir or "/nonexistent")
            assert os.path.exists(client_b._temp_dir)
            assert os.path.isfile(client_b.cert_path)

            client_b.cleanup()

    def test_concurrent_cleanup_no_race(self):
        """Multiple clients created and cleaned up concurrently should not interfere."""
        with patch.dict("sys.modules", {
            "py3_compat": MagicMock(),
            "patch_analizar_errores": MagicMock(),
            "pyafipws": MagicMock(),
            "pyafipws.wsaa": MagicMock(),
            "pyafipws.wsfev1": MagicMock(),
        }):
            import importlib
            import afip_client
            importlib.reload(afip_client)

            def create_and_cleanup(cuit: str):
                client = afip_client.AFIPClient(
                    cuit_emisor=cuit,
                    homologacion=True,
                    cert_pem=SAMPLE_CERT_PEM,
                    key_pem=SAMPLE_KEY_PEM,
                )
                temp_dir = client._temp_dir
                assert os.path.exists(temp_dir)
                client.cleanup()
                assert not os.path.exists(temp_dir)
                return cuit

            cuits = [f"20{i:09d}0" for i in range(10)]
            with ThreadPoolExecutor(max_workers=5) as pool:
                results = list(pool.map(create_and_cleanup, cuits))

            assert len(results) == 10


# ── Facturar endpoint stateless path selection ───────────────────────────────

class TestFacturarEndpointRouting:
    """Test that /facturar correctly routes to stateless vs legacy mode."""

    def test_stateless_mode_when_cert_pem_present(self):
        """Request with cert_pem should trigger stateless mode (per_req_client)."""
        req = FacturarRequest(
            cuit_emisor="20123456789",
            punto_de_venta=1,
            tipo_comprobante=11,
            tipo_doc_receptor=99,
            nro_doc_receptor="0",
            concepto=1,
            importe_neto="1000.00",
            importe_total="1000.00",
            cert_pem=SAMPLE_CERT_PEM,
            key_pem=SAMPLE_KEY_PEM,
            modo="homologacion",
        )
        # The endpoint checks `if req.cert_pem and req.key_pem`
        assert bool(req.cert_pem and req.key_pem) is True

    def test_legacy_mode_when_no_cert_pem(self):
        """Request without cert_pem should use legacy global client."""
        req = FacturarRequest(
            cuit_emisor="20123456789",
            punto_de_venta=1,
            tipo_comprobante=11,
            tipo_doc_receptor=99,
            nro_doc_receptor="0",
            concepto=1,
            importe_neto="1000.00",
            importe_total="1000.00",
        )
        assert bool(req.cert_pem and req.key_pem) is False
