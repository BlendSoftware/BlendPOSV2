// ─────────────────────────────────────────────────────────────────────────────
// API Client — BlendPOS
// Cliente HTTP centralizado. Cuando el backend Go esté disponible,
// configurar VITE_API_URL en .env y reemplazar las funciones mock de
// cada módulo por llamadas a apiClient.get/post/put/delete.
// ─────────────────────────────────────────────────────────────────────────────

import { tokenStore } from '../store/tokenStore';

// VITE_API_BASE debe apuntar al backend Go, SIN path final (ej: http://localhost:8000)
const BASE_URL = (import.meta.env.VITE_API_BASE as string | undefined) ?? 'http://localhost:8000';

// ── Typed network error for offline detection ────────────────────────────────
export class OfflineError extends Error {
    constructor() {
        super('Sin conexión a internet');
        this.name = 'OfflineError';
    }
}

// ── Auto-refresh shared state (B-03) ────────────────────────────────────────
// Prevents multiple concurrent refresh attempts when several requests
// receive 401 at the same time.

/** Shape returned by the backend refresh endpoint. */
export interface RefreshResult {
    access_token: string;
    refresh_token: string;
    user?: {
        id: string;
        username: string;
        nombre: string;
        rol: string;
        punto_de_venta: number | null;
        sucursal_id: string | null;
        sucursal_nombre: string | null;
    };
}

let _refreshPromiseTyped: Promise<RefreshResult> | null = null;

/**
 * Attempts to obtain a new access token using the stored refresh token.
 * Only ONE refresh request is in-flight at any time; concurrent callers
 * share the same promise.
 *
 * Uses raw fetch() — MUST NOT go through `request()` / apiClient to avoid
 * the 401 interceptor triggering a recursive refresh loop.
 */
export async function refreshAccessToken(): Promise<RefreshResult> {
    if (_refreshPromiseTyped) return _refreshPromiseTyped;

    _refreshPromiseTyped = (async () => {
        const refreshToken = tokenStore.getRefreshToken();
        if (!refreshToken) throw new Error('No refresh token available');

        const res = await fetch(`${BASE_URL}/v1/auth/refresh`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: refreshToken }),
        });

        if (!res.ok) throw new Error('Refresh failed');

        const data = (await res.json()) as RefreshResult;
        tokenStore.setTokens(data.access_token, data.refresh_token);

        // Schedule proactive refresh before new access token expires.
        scheduleProactiveRefresh(data.access_token);

        return data;
    })();

    try {
        return await _refreshPromiseTyped;
    } finally {
        _refreshPromiseTyped = null;
    }
}

// ── Proactive refresh timer (B-03 Option B) ─────────────────────────────────
let _refreshTimer: ReturnType<typeof setTimeout> | null = null;

/**
 * Decodes the JWT payload (without verifying signature — that's the
 * backend's job) and schedules a silent refresh 60 s before expiry.
 */
export function scheduleProactiveRefresh(accessToken: string): void {
    if (_refreshTimer) clearTimeout(_refreshTimer);

    try {
        const payload = JSON.parse(atob(accessToken.split('.')[1]));
        const exp = (payload.exp as number) * 1000; // ms
        const msUntilRefresh = exp - Date.now() - 60_000; // 60 s before

        if (msUntilRefresh <= 0) return; // already very close; reactive path will handle it

        _refreshTimer = setTimeout(() => {
            refreshAccessToken().catch(() => {
                // Proactive refresh failed — will be retried reactively on next 401.
            });
        }, msUntilRefresh);
    } catch {
        // Malformed token — ignore; reactive path will handle it.
    }
}

/** Cancel any pending proactive refresh (call on logout). */
export function cancelProactiveRefresh(): void {
    if (_refreshTimer) {
        clearTimeout(_refreshTimer);
        _refreshTimer = null;
    }
}

// Read the JWT access token from the in-memory store (P1-003).
// Tokens are never written to localStorage — this function can only return
// a value if the user has logged in during the current page session.
function getToken(): string | null {
    return tokenStore.getAccessToken();
}

// Read selected sucursal ID from localStorage (persisted by useSucursalStore's
// Zustand persist middleware). Returns null when "Todas las sucursales" is
// selected. Avoids importing useSucursalStore directly to prevent a circular
// dependency chain: client → useSucursalStore → sucursales API → client.
function getSucursalId(): string | null {
    try {
        const raw = localStorage.getItem('blendpos-sucursal');
        if (!raw) return null;
        const parsed = JSON.parse(raw);
        return parsed?.state?.sucursalId ?? null;
    } catch {
        return null;
    }
}

type QueryParams = Record<string, string | number | boolean | undefined | null>;

async function request<T>(
    path: string,
    options: RequestInit & { params?: QueryParams } = {},
    _isRetry = false,
): Promise<T> {
    const { params, ...init } = options;

    let url = `${BASE_URL}${path}`;
    if (params) {
        const search = new URLSearchParams();
        for (const [k, v] of Object.entries(params)) {
            if (v !== undefined && v !== null) search.set(k, String(v));
        }
        const q = search.toString();
        if (q) url += `?${q}`;
    }

    const token = getToken();
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...(init.headers as Record<string, string> | undefined ?? {}),
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    // ── Sucursal context injection ──────────────────────────────────────
    // Automatically sends the selected sucursal so backend endpoints can
    // filter by branch without each page having to pass it explicitly.
    // Read from localStorage (persisted by useSucursalStore) to avoid a
    // circular import: client → useSucursalStore → sucursales API → client.
    const sucursalId = getSucursalId();
    if (sucursalId) headers['X-Sucursal-Id'] = sucursalId;

    // ── Offline-resilient fetch ──────────────────────────────────────────
    let response: Response;
    try {
        response = await fetch(url, { ...init, headers });
    } catch (err) {
        // TypeError: Failed to fetch → network is down
        if (!navigator.onLine || (err instanceof TypeError && /fetch|network/i.test(err.message))) {
            throw new OfflineError();
        }
        throw err;
    }

    if (response.status === 401) {
        // ── B-03: Try silent refresh before giving up ────────────────────
        // Attempt refresh if:
        //   - We had an access token (normal expiry), OR
        //   - We have a refresh token in sessionStorage (page reload — access
        //     token was in memory only and is gone, but refresh survived).
        // Never retry more than once (prevents infinite loop).
        const hasRefreshToken = !!tokenStore.getRefreshToken();
        if (!_isRetry && (token || hasRefreshToken)) {
            try {
                await refreshAccessToken();
                // Retry the original request with the fresh token.
                return request<T>(path, options, true);
            } catch {
                // Refresh failed — session truly expired.
            }
        }

        // Clear auth state — ProtectedRoute will handle the redirect.
        if (token || hasRefreshToken) {
            cancelProactiveRefresh();
            tokenStore.clearTokens();
            // Lazy import to avoid circular dependency (useAuthStore → apiClient → useAuthStore).
            // Only reset local state; do NOT call logout() which would hit apiClient again.
            import('../store/useAuthStore').then(({ useAuthStore }) => {
                useAuthStore.setState({
                    user: null,
                    isAuthenticated: false,
                    tenantId: null,
                    mustChangePassword: false,
                });
            });
        }
        throw new Error('Sesión expirada o no autorizado.');
    }

    if (!response.ok) {
        const body = await response.text();
        throw new Error(`${response.status} ${response.statusText}: ${body}`);
    }

    // 204 No Content
    if (response.status === 204) return undefined as T;

    return response.json() as Promise<T>;
}

export const apiClient = {
    get: <T>(path: string, params?: QueryParams) =>
        request<T>(path, { method: 'GET', params }),

    post: <T>(path: string, body: unknown) =>
        request<T>(path, { method: 'POST', body: JSON.stringify(body) }),

    put: <T>(path: string, body: unknown) =>
        request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),

    patch: <T>(path: string, body: unknown) =>
        request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),

    delete: <T>(path: string, body?: unknown) =>
        request<T>(path, {
            method: 'DELETE',
            ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
        }),
};
