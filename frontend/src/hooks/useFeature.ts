import { useCallback, useRef, useSyncExternalStore } from 'react';
import { obtenerPlanActual } from '../services/api/tenant';
import { useAuthStore } from '../store/useAuthStore';

/**
 * Module-level cache for plan features. Shared across all hook instances.
 */
let cachedFeatures: Record<string, boolean> | null = null;
let cachedForTenant: string | null = null;
let fetchPromise: Promise<void> | null = null;
let listeners: Array<() => void> = [];
let revision = 0;

function notify() {
    revision++;
    for (const l of listeners) l();
}

function subscribe(listener: () => void) {
    listeners.push(listener);
    return () => { listeners = listeners.filter((l) => l !== listener); };
}

function ensureFeatures(tenantId: string | null) {
    if (!tenantId) return;
    if (cachedFeatures && cachedForTenant === tenantId) return;
    if (fetchPromise && cachedForTenant === tenantId) return;

    // Invalidate on tenant change
    if (cachedForTenant !== tenantId) {
        cachedFeatures = null;
        cachedForTenant = tenantId;
    }

    fetchPromise = obtenerPlanActual()
        .then((plan) => {
            cachedFeatures = plan.features ?? {};
            cachedForTenant = tenantId;
            fetchPromise = null;
            notify();
        })
        .catch(() => {
            cachedFeatures = {};
            cachedForTenant = tenantId;
            fetchPromise = null;
            notify();
        });
}

interface FeatureSnapshot { enabled: boolean; loading: boolean }

/**
 * useFeature — checks if a plan feature flag is enabled for the current tenant.
 *
 * @example
 * const { enabled, loading } = useFeature('analytics_avanzados');
 */
export function useFeature(feature: string): FeatureSnapshot {
    const tenantId = useAuthStore((s) => s.tenantId);
    const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

    // Kick off fetch if needed (side-effect-free: only starts a promise)
    if (isAuthenticated && tenantId) {
        ensureFeatures(tenantId);
    }

    // Stable snapshot ref — useSyncExternalStore requires Object.is equality
    const prevRef = useRef<{ enabled: boolean; loading: boolean; rev: number }>({ enabled: false, loading: false, rev: -1 });

    const getSnapshot = useCallback((): FeatureSnapshot => {
        let enabled = false;
        let loading = false;

        if (!isAuthenticated || !tenantId) {
            // defaults: enabled=false, loading=false
        } else if (cachedFeatures && cachedForTenant === tenantId) {
            enabled = cachedFeatures[feature] ?? false;
        } else {
            loading = true;
        }

        const prev = prevRef.current;
        if (prev.rev === revision && prev.enabled === enabled && prev.loading === loading) {
            return prev;
        }

        const next = { enabled, loading, rev: revision };
        prevRef.current = next;
        return next;
    }, [feature, tenantId, isAuthenticated]);

    const snapshot = useSyncExternalStore(subscribe, getSnapshot);
    return { enabled: snapshot.enabled, loading: snapshot.loading };
}

/**
 * Invalidates the cached features. Call after plan upgrade/downgrade.
 */
export function invalidateFeatureCache(): void {
    cachedFeatures = null;
    cachedForTenant = null;
    fetchPromise = null;
    notify();
}
