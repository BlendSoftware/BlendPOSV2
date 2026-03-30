// ─────────────────────────────────────────────────────────────────────────────
// useSucursalRefresh — Triggers a callback whenever the selected sucursal changes.
//
// Usage:
//   useSucursalRefresh(fetchData);
//
// The callback fires:
//   1. On mount (initial load)
//   2. Every time the user switches sucursal in the admin panel
// ─────────────────────────────────────────────────────────────────────────────

import { useEffect, useRef } from 'react';
import { useSucursalStore } from '../store/useSucursalStore';

/**
 * Calls `onRefresh` whenever the selected sucursal changes.
 * Skips the initial mount call if `skipInitial` is true (default: false).
 */
export function useSucursalRefresh(onRefresh: () => void, skipInitial = false): void {
    const switchCounter = useSucursalStore((s) => s._switchCounter);
    const isFirstRender = useRef(true);

    useEffect(() => {
        if (isFirstRender.current) {
            isFirstRender.current = false;
            if (skipInitial) return;
        }
        onRefresh();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [switchCounter]);
}
