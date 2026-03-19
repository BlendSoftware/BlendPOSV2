import 'fake-indexeddb/auto';
import '@testing-library/jest-dom/vitest';

// ── jsdom polyfills required by Mantine / testing-library ────────────────────

// Mantine requires window.matchMedia — jsdom does not implement it.
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
    }),
});

// Mantine Stepper and other components use ResizeObserver.
class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
}
globalThis.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver;
