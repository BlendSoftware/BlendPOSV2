import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useSyncStatus } from './useSyncStatus';

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('../offline/sync', () => ({
    getSyncStats: vi.fn(async () => ({ pending: 0, error: 0 })),
    trySyncQueue: vi.fn(async () => undefined),
}));

vi.mock('@mantine/notifications', () => ({
    notifications: { show: vi.fn() },
}));

describe('useSyncStatus', () => {
    beforeEach(() => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        vi.clearAllMocks();
        // Default: online
        Object.defineProperty(navigator, 'onLine', { value: true, writable: true, configurable: true });
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it('returns initial state with zero pending', async () => {
        const { result } = renderHook(() => useSyncStatus());

        // Initial state — pending/error are 0, syncState may be 'syncing'
        // because the hook immediately kicks off an attemptSync on mount.
        expect(result.current.pending).toBe(0);
        expect(result.current.error).toBe(0);
        expect(['idle', 'syncing']).toContain(result.current.syncState);
    });

    it('reflects pending count from getSyncStats', async () => {
        const { getSyncStats } = await import('../offline/sync');
        vi.mocked(getSyncStats).mockResolvedValue({ pending: 5, error: 0 });

        const { result } = renderHook(() => useSyncStatus());

        // Stats polling runs every 2s
        await act(async () => {
            vi.advanceTimersByTime(2500);
        });

        await waitFor(() => {
            expect(result.current.pending).toBe(5);
        });
    });

    it('returns error count from getSyncStats', async () => {
        const { getSyncStats } = await import('../offline/sync');
        vi.mocked(getSyncStats).mockResolvedValue({ pending: 2, error: 3 });

        const { result } = renderHook(() => useSyncStatus());

        await act(async () => {
            vi.advanceTimersByTime(2500);
        });

        await waitFor(() => {
            expect(result.current.pending).toBe(2);
            expect(result.current.error).toBe(3);
        });
    });
});
