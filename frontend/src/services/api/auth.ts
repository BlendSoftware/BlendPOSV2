// ─────────────────────────────────────────────────────────────────────────────
// Auth API — conecta con POST /v1/auth/login y POST /v1/auth/refresh
// Mapea los DTOs del backend Go exactamente.
// ─────────────────────────────────────────────────────────────────────────────

import { apiClient } from '../../api/client';

// ── Response Types ────────────────────────────────────────────────────────────

export interface UsuarioResponse {
    id: string;
    username: string;
    nombre: string;
    rol: 'cajero' | 'supervisor' | 'administrador';
    punto_de_venta: number | null;
    sucursal_id: string | null;
}

export interface LoginResponse {
    access_token: string;
    refresh_token: string;
    token_type: string;
    expires_in: number;
    user: UsuarioResponse;
    must_change_password: boolean;
}

// ── API Calls ─────────────────────────────────────────────────────────────────

const BASE_URL = (import.meta.env.VITE_API_BASE as string | undefined) ?? 'http://localhost:8000';

/**
 * POST /v1/auth/login
 * Autentica al usuario y retorna tokens JWT.
 */
export async function loginApi(username: string, password: string): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>('/v1/auth/login', { username, password });
}

/**
 * POST /v1/auth/refresh
 * Renueva el access token usando el refresh token.
 *
 * IMPORTANT: Uses raw fetch() — MUST NOT go through apiClient to avoid the
 * 401 interceptor triggering a recursive refresh loop. The deduplication lock
 * is handled by the caller (refreshAccessToken in client.ts or useAuthStore).
 */
export async function refreshApi(refreshToken: string): Promise<LoginResponse> {
    const res = await fetch(`${BASE_URL}/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) {
        throw new Error(`Refresh failed: ${res.status}`);
    }

    return res.json() as Promise<LoginResponse>;
}

/**
 * POST /v1/auth/change-password
 * SEC-03: Cambia contraseña obligatoria tras primer login.
 */
export async function changePasswordApi(newPassword: string): Promise<void> {
    await apiClient.post<void>('/v1/auth/change-password', { new_password: newPassword });
}

/**
 * POST /v1/auth/logout
 * Revoca el access token actual en el servidor (agrega su jti a la blocklist de Redis).
 * Best-effort: los errores no bloquean el logout local.
 */
export async function logoutApi(): Promise<void> {
    await apiClient.post<void>('/v1/auth/logout', {});
}
