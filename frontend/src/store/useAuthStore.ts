// ─────────────────────────────────────────────────────────────────────────────
// Auth Store — Zustand con persistencia parcial en localStorage.
//
// Los tokens JWT se guardan ÚNICAMENTE en memoria (tokenStore) para reducir
// la superficie de ataque XSS (P1-003).  Solo el perfil del usuario y el
// flag isAuthenticated se persisten en localStorage para restaurar la UI
// tras un hard-refresh (el token se obtiene de nuevo con silent-refresh).
// ─────────────────────────────────────────────────────────────────────────────

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { IUser, Rol } from '../types';
import { loginApi } from '../services/api/auth';
import { apiClient, scheduleProactiveRefresh, cancelProactiveRefresh, refreshAccessToken } from '../api/client';
import { tokenStore } from './tokenStore';
import { useSucursalStore } from './useSucursalStore';

// ── Usuarios demo — SOLO desarrollo (P1-004) ──────────────────────────────────
// La constante es accesible únicamente cuando el bundler incluye el bloque
// import.meta.env.DEV === true.  En producción se genera un módulo vacío.
// Las credenciales se leen de .env (VITE_DEMO_PASS) — nunca hardcodeadas.

const DEMO_PASS = (import.meta.env.DEV && import.meta.env.VITE_DEMO_PASS as string) || '';

const DEMO_USERS: (IUser & { password: string; username: string })[] = import.meta.env.DEV && DEMO_PASS
    ? [
        { id: 'u1', nombre: 'Carlos Administrador', email: 'admin@blendpos.com', rol: 'admin', activo: true, creadoEn: '2025-01-10T10:00:00Z', username: 'admin', password: DEMO_PASS },
        { id: 'u2', nombre: 'María Supervisora', email: 'super@blendpos.com', rol: 'supervisor', activo: true, creadoEn: '2025-02-01T10:00:00Z', username: 'supervisor', password: DEMO_PASS },
        { id: 'u3', nombre: 'Juan Cajero', email: 'caja@blendpos.com', rol: 'cajero', activo: true, creadoEn: '2025-03-15T10:00:00Z', username: 'cajero', password: DEMO_PASS },
    ]
    : [];

/**
 * Extracts the tenant_id (claim "tid") from a JWT without verifying the
 * signature — the server already validated the token. Used only to populate
 * the client-side tenantId state for offline_id generation.
 */
function extractTenantIdFromToken(token: string): string | null {
    try {
        const payload = token.split('.')[1];
        if (!payload) return null;
        const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
        return typeof decoded.tid === 'string' ? decoded.tid : null;
    } catch {
        return null;
    }
}

// El backend usa 'administrador', el frontend usa 'admin'
function mapRol(backendRol: string): Rol {
    if (backendRol === 'administrador') return 'admin';
    if (backendRol === 'supervisor') return 'supervisor';
    if (backendRol === 'superadmin') return 'superadmin';
    return 'cajero';
}

interface AuthState {
    user: IUser | null;
    isAuthenticated: boolean;
    /** Multi-tenant: UUID of the tenant this session belongs to */
    tenantId: string | null;
    /** SEC-03: true when the backend requires a password change on first login */
    mustChangePassword: boolean;
    _hasHydrated: boolean;

    login: (usernameOrEmail: string, password: string) => Promise<boolean>;
    logout: () => Promise<void>;
    refresh: () => Promise<boolean>;
    /** Called on app mount to silently restore the session via refresh token. */
    initAuth: () => Promise<void>;
    hasRole: (roles: Rol[]) => boolean;
    /** SEC-03: Called after user completes forced password change */
    clearMustChangePassword: () => void;
}

export const useAuthStore = create<AuthState>()(
    persist(
        (set, get) => ({
            user: null,
            isAuthenticated: false,
            tenantId: null,
            mustChangePassword: false,
            _hasHydrated: false,

            login: async (usernameOrEmail, password) => {
                const backendAvailable = !!(import.meta.env.VITE_API_BASE as string | undefined);

                if (backendAvailable) {
                    try {
                        const resp = await loginApi(usernameOrEmail, password);
                        const u = resp.user;
                        const user: IUser = {
                            id: u.id,
                            nombre: u.nombre,
                            email: usernameOrEmail.includes('@') ? usernameOrEmail : '',
                            rol: mapRol(u.rol),
                            activo: true,
                            creadoEn: new Date().toISOString(),
                            puntoDeVenta: u.punto_de_venta ?? undefined,
                            sucursalId: u.sucursal_id ?? undefined,
                        };
                        // Store tokens in memory only — never in localStorage
                        tokenStore.setTokens(resp.access_token, resp.refresh_token);
                        scheduleProactiveRefresh(resp.access_token);
                        // Extract tenant_id from JWT payload (claim "tid")
                        const tenantId = extractTenantIdFromToken(resp.access_token);
                        set({ user, isAuthenticated: true, tenantId, mustChangePassword: resp.must_change_password ?? false });
                        return true;
                    } catch (err) {
                        // Re-throw network/server errors so the UI can show appropriate messages.
                        // Only swallow 401-type errors (bad credentials) by returning false.
                        if (err instanceof Error && (err.name === 'OfflineError' || /^5\d{2}\s/.test(err.message) || /fetch|network/i.test(err.message))) {
                            throw err;
                        }
                        return false;
                    }
                }

                // Fallback demo (sin backend) — dev only
                if (!import.meta.env.DEV) return false;
                await new Promise((r) => setTimeout(r, 400));
                const found = DEMO_USERS.find(
                    (u) => (u.email === usernameOrEmail || u.username === usernameOrEmail) &&
                        u.password === password && u.activo,
                );
                if (!found) return false;
                // eslint-disable-next-line @typescript-eslint/no-unused-vars
                const { password: _pw, username: _un, ...user } = found;
                const fakeToken = btoa(JSON.stringify({ sub: user.id, rol: user.rol, exp: Date.now() + 28800_000 }));
                tokenStore.setTokens(fakeToken, '');
                set({ user, isAuthenticated: true });
                return true;
            },

            logout: async () => {
                // Attempt server-side revocation (best-effort, don't block UI)
                try {
                    const accessToken = tokenStore.getAccessToken();
                    if (accessToken) {
                        await apiClient.post('/v1/auth/logout', {});
                    }
                } catch {
                    // Logout is best-effort — clear local state regardless
                }
                cancelProactiveRefresh();
                tokenStore.clearTokens();
                // Clear sucursal selection to prevent multi-tenant data leak
                useSucursalStore.getState().setSucursal(null, null);
                set({ user: null, isAuthenticated: false, tenantId: null, mustChangePassword: false });
            },

            refresh: async () => {
                const refreshToken = tokenStore.getRefreshToken();
                if (!refreshToken) return false;
                try {
                    // Uses the shared refreshAccessToken which:
                    // 1) Deduplicates concurrent calls via _refreshPromiseTyped
                    // 2) Uses raw fetch() to bypass the 401 interceptor
                    // 3) Stores new tokens + schedules proactive refresh
                    const result = await refreshAccessToken();
                    const tenantId = extractTenantIdFromToken(result.access_token);
                    if (result.user) {
                        const u = result.user;
                        const user: IUser = {
                            id: u.id,
                            nombre: u.nombre,
                            email: '',
                            rol: mapRol(u.rol),
                            activo: true,
                            creadoEn: new Date().toISOString(),
                            sucursalId: u.sucursal_id ?? undefined,
                        };
                        set({ user, isAuthenticated: true, tenantId });
                    } else {
                        // No user in response — keep existing user, just update tenantId
                        set({ isAuthenticated: true, tenantId });
                    }
                    return true;
                } catch {
                    cancelProactiveRefresh();
                    tokenStore.clearTokens();
                    set({ user: null, isAuthenticated: false, tenantId: null });
                    return false;
                }
            },

            /**
             * Called once on app mount (App.tsx useEffect).
             * If the store says the user was authenticated, try a silent token
             * refresh so they don't have to log in again after a page reload.
             */
            initAuth: async () => {
                if (!get().isAuthenticated) return;
                // Token is gone (page reload) — try refresh
                if (!tokenStore.getAccessToken()) {
                    const ok = await get().refresh();
                    if (!ok) {
                        // Couldn't restore token (no refresh token in memory or API error).
                        // Clear local auth state so ProtectedRoute redirects to login.
                        tokenStore.clearTokens();
                        set({ user: null, isAuthenticated: false, tenantId: null });
                    }
                }
            },

            hasRole: (roles) => {
                const { user } = get();
                return user !== null && roles.includes(user.rol);
            },

            clearMustChangePassword: () => set({ mustChangePassword: false }),
        }),
        {
            name: 'blendpos-auth',
            partialize: (state) => ({
                user: state.user,
                isAuthenticated: state.isAuthenticated,
                tenantId: state.tenantId,
            }),
            onRehydrateStorage: () => (_state, error) => {
                if (error) {
                    console.warn('[useAuthStore] Hydration error, clearing corrupt state:', error);
                }
                queueMicrotask(() => useAuthStore.setState({ _hasHydrated: true }));
            },
        }
    )
);

// ── Safety net: if onRehydrateStorage never fires (edge case with some
// Zustand versions or SSR), force _hasHydrated after a short delay. ──────────
if (typeof window !== 'undefined') {
    setTimeout(() => {
        if (!useAuthStore.getState()._hasHydrated) {
            console.warn('[useAuthStore] Hydration timeout — forcing _hasHydrated = true');
            useAuthStore.setState({ _hasHydrated: true });
        }
    }, 500);
}
