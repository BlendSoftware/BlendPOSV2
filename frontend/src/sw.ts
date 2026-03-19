/// <reference lib="webworker" />

/**
 * Custom Service Worker — BlendPOS
 *
 * Responsabilidades:
 * 1. Pre-caché de assets (inyectado por vite-plugin-pwa con injectManifest).
 * 2. Runtime caching: network-first para API, cache-first para assets estáticos.
 * 3. Offline navigation fallback: devuelve index.html cacheado para rutas SPA.
 * 4. Background Sync: cuando el navegador recupera conectividad, el SW
 *    notifica a la pestaña activa para que ejecute trySyncQueue().
 * 5. SKIP_WAITING: acepta mensaje del main thread para activar nueva versión.
 */

import { precacheAndRoute, cleanupOutdatedCaches, createHandlerBoundToURL } from 'workbox-precaching';
import { registerRoute, NavigationRoute } from 'workbox-routing';
import { NetworkFirst, CacheFirst, StaleWhileRevalidate } from 'workbox-strategies';
import { ExpirationPlugin } from 'workbox-expiration';
import { CacheableResponsePlugin } from 'workbox-cacheable-response';

declare const self: ServiceWorkerGlobalScope & {
    __WB_MANIFEST: Array<{ url: string; revision: string | null }>;
    addEventListener(type: 'sync', listener: (event: ExtendableEvent & { tag: string }) => void, options?: boolean | AddEventListenerOptions): void;
};

// ── Precache ──────────────────────────────────────────────────────────────────
// Inyecta y cachea el app shell (HTML, JS, CSS) generado por Vite en build time.
precacheAndRoute(self.__WB_MANIFEST);
cleanupOutdatedCaches();

// ── Offline Navigation Fallback ──────────────────────────────────────────────
// Para cualquier navegación (rutas SPA), devolver index.html cacheado.
// Esto permite que la app funcione offline: React Router maneja las rutas.
const handler = createHandlerBoundToURL('/index.html');
const navigationRoute = new NavigationRoute(handler, {
    // No interceptar requests a la API ni a assets con extensión explícita
    denylist: [/^\/v1\//, /^\/api\//, /\.\w+$/],
});
registerRoute(navigationRoute);

// ── Runtime Caching: API calls ───────────────────────────────────────────────
// Network-first: intenta la red, si falla usa caché (útil para consultas de
// precios o catálogo cuando se pierde conexión brevemente).
registerRoute(
    ({ url }) => url.pathname.startsWith('/v1/'),
    new NetworkFirst({
        cacheName: 'api-cache',
        networkTimeoutSeconds: 5,
        plugins: [
            new CacheableResponsePlugin({ statuses: [0, 200] }),
            new ExpirationPlugin({
                maxEntries: 200,
                maxAgeSeconds: 60 * 60, // 1 hora
            }),
        ],
    }),
);

// ── Runtime Caching: Google Fonts (si se usan) ───────────────────────────────
registerRoute(
    ({ url }) => url.origin === 'https://fonts.googleapis.com' || url.origin === 'https://fonts.gstatic.com',
    new StaleWhileRevalidate({
        cacheName: 'google-fonts',
        plugins: [
            new CacheableResponsePlugin({ statuses: [0, 200] }),
            new ExpirationPlugin({ maxEntries: 30 }),
        ],
    }),
);

// ── Runtime Caching: Images ──────────────────────────────────────────────────
// Cache-first: imágenes cambian poco, priorizamos velocidad.
registerRoute(
    ({ request }) => request.destination === 'image',
    new CacheFirst({
        cacheName: 'images-cache',
        plugins: [
            new CacheableResponsePlugin({ statuses: [0, 200] }),
            new ExpirationPlugin({
                maxEntries: 100,
                maxAgeSeconds: 30 * 24 * 60 * 60, // 30 días
            }),
        ],
    }),
);

// ── Runtime Caching: Fonts ───────────────────────────────────────────────────
// Cache-first: las fuentes casi nunca cambian.
registerRoute(
    ({ request }) => request.destination === 'font',
    new CacheFirst({
        cacheName: 'fonts-cache',
        plugins: [
            new CacheableResponsePlugin({ statuses: [0, 200] }),
            new ExpirationPlugin({
                maxEntries: 20,
                maxAgeSeconds: 365 * 24 * 60 * 60, // 1 año
            }),
        ],
    }),
);

// ── Background Sync ───────────────────────────────────────────────────────────
/**
 * 'sync' event: disparado por el browser cuando la conexión se restaura
 * y el tag 'blendpos-sync-ventas' está pendiente.
 *
 * Estrategia: postMessage a todos los clientes (pestañas) abiertos para
 * que ejecuten trySyncQueue() en el main thread donde IndexedDB está disponible.
 */
self.addEventListener('sync', (event) => {
    if (event.tag === 'blendpos-sync-ventas') {
        event.waitUntil(notifyClientsToSync());
    }
});

async function notifyClientsToSync(): Promise<void> {
    const clients = await self.clients.matchAll({
        includeUncontrolled: true,
        type: 'window',
    });
    for (const client of clients) {
        client.postMessage({ type: 'SYNC_SALES' });
    }
}

// ── SKIP_WAITING ─────────────────────────────────────────────────────────────
// Cuando el main thread envía SKIP_WAITING, activar la nueva versión del SW
// inmediatamente en vez de esperar a que se cierren todas las pestañas.
self.addEventListener('message', (event) => {
    if (event.data?.type === 'SKIP_WAITING') {
        void self.skipWaiting();
    }
});

// ── Activate ──────────────────────────────────────────────────────────────────
// Toma control inmediato de todos los clientes sin esperar recarga.
self.addEventListener('activate', (event) => {
    event.waitUntil(self.clients.claim());
});
