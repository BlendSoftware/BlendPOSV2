"""
Módulo de compatibilidad Python 2 -> 3 para pyafipws.
Se importa al inicio para monkey-patch funciones que requieren bytes en Python 3.
"""
import hashlib
import sys

# Guardar las funciones originales
_orig_md5 = hashlib.md5
_orig_sha1 = hashlib.sha1
_orig_sha256 = hashlib.sha256
_orig_sha512 = hashlib.sha512


def _make_hash_wrapper(original_func):
    """Crea un wrapper que convierte strings a bytes automáticamente."""
    def wrapper(data=b'', **kwargs):
        # Si data es string, convertir a bytes
        if isinstance(data, str):
            data = data.encode('utf-8')
        return original_func(data, **kwargs)
    return wrapper


# Reemplazar las funciones de hashlib con versiones que aceptan strings
hashlib.md5 = _make_hash_wrapper(_orig_md5)
hashlib.sha1 = _make_hash_wrapper(_orig_sha1)
hashlib.sha256 = _make_hash_wrapper(_orig_sha256)
hashlib.sha512 = _make_hash_wrapper(_orig_sha512)


# Monkey-patch para métodos .update() de objetos hash
class HashWrapper:
    """Wrapper para objetos hash que convierte strings a bytes en .update()"""
    def __init__(self, hash_obj):
        self._hash_obj = hash_obj
    
    def update(self, data):
        if isinstance(data, str):
            data = data.encode('utf-8')
        return self._hash_obj.update(data)
    
    def __getattr__(self, name):
        # Delegar todos los demás métodos al objeto original
        return getattr(self._hash_obj, name)


# Parchear los constructores para retornar wrappers
_orig_md5_unwrapped = hashlib.md5
_orig_sha1_unwrapped = hashlib.sha1
_orig_sha256_unwrapped = hashlib.sha256
_orig_sha512_unwrapped = hashlib.sha512


def _wrapped_md5(data=b'', **kwargs):
    if isinstance(data, str):
        data = data.encode('utf-8')
    return HashWrapper(_orig_md5(data, **kwargs))


def _wrapped_sha1(data=b'', **kwargs):
    if isinstance(data, str):
        data = data.encode('utf-8')
    return HashWrapper(_orig_sha1(data, **kwargs))


def _wrapped_sha256(data=b'', **kwargs):
    if isinstance(data, str):
        data = data.encode('utf-8')
    return HashWrapper(_orig_sha256(data, **kwargs))


def _wrapped_sha512(data=b'', **kwargs):
    if isinstance(data, str):
        data = data.encode('utf-8')
    return HashWrapper(_orig_sha512(data, **kwargs))


hashlib.md5 = _wrapped_md5
hashlib.sha1 = _wrapped_sha1
hashlib.sha256 = _wrapped_sha256
hashlib.sha512 = _wrapped_sha512

print("[py3_compat] Monkey-patch de hashlib aplicado para compatibilidad Python 2->3", file=sys.stderr)

# --- PARCHE SSL SECLEVEL=0 PARA AFIP (DH_KEY_TOO_SMALL) ---
#
# WHY THIS IS GLOBAL: AFIP's WSAA and WSFEV1 SOAP endpoints use legacy TLS
# parameters (small DH keys) that OpenSSL 3.x rejects at SECLEVEL>=1. The
# pyafipws library uses Python's default SSL context internally (via
# pysimplesoap → httplib2/urllib) and does NOT expose an API to inject a
# custom SSL context per-connection.
#
# Scoping SECLEVEL=0 to only AFIP connections would require monkey-patching
# deep inside pysimplesoap's HTTP transport layer, which is fragile and
# breaks across pyafipws versions. Since this sidecar is a single-purpose
# container that ONLY talks to AFIP (no other outbound HTTPS), the global
# override is acceptable.
#
# RISK: If this container ever makes HTTPS calls to non-AFIP services,
# those connections will also use SECLEVEL=0 (weaker cipher negotiation).
# Mitigate by keeping this sidecar single-purpose — only AFIP traffic.
#
# The Dockerfile also sets SECLEVEL=0 in openssl.cnf as a belt-and-suspenders
# measure, since some pyafipws code paths use subprocess openssl commands.

import ssl

_orig_create_default_context = ssl.create_default_context

def _legacy_default_context(*args, **kwargs):
    ctx = _orig_create_default_context(*args, **kwargs)
    ctx.set_ciphers('DEFAULT@SECLEVEL=0')
    return ctx

# Replace default context creators so ALL SSL connections in this process
# use SECLEVEL=0. See comment above for why this must be global.
ssl.create_default_context = _legacy_default_context
if hasattr(ssl, '_create_default_https_context'):
    ssl._create_default_https_context = _legacy_default_context
# --------------------------------------------------