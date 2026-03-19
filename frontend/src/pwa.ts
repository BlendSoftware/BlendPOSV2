import { registerSW } from 'virtual:pwa-register';
import { notifications } from '@mantine/notifications';
import { trySyncQueue } from './offline/sync';

// ── SW Registration + Update Flow ────────────────────────────────────────────
// registerType:'autoUpdate' en vite.config.ts + immediate:true acá hacen que
// el SW se registre al cargar. Cuando hay una nueva versión:
// 1. onNeedRefresh se dispara → mostramos notificación "Nueva versión disponible"
// 2. El usuario hace click en "Actualizar" → llamamos updateSW() que envía
//    SKIP_WAITING al SW waiting y luego recarga la página.
const updateSW = registerSW({
    immediate: true,
    onNeedRefresh() {
        notifications.show({
            id: 'pwa-update',
            title: 'Nueva versión disponible',
            message: 'Hay una actualización de BlendPOS lista.',
            color: 'blue',
            autoClose: false,
            withCloseButton: true,
            // Mantine notifications no soporta `actions` directamente,
            // usamos onClick para disparar la actualización al cerrar.
            // Como alternativa, mostramos la notificación con un handler global.
            onClose: () => {
                // Si el usuario cierra sin actualizar, no hacemos nada.
                // La actualización se aplicará cuando cierre todas las pestañas.
            },
        });

        // Agregar el botón "Actualizar" al DOM de la notificación.
        // Usamos un approach más simple: reemplazamos la notificación con una
        // que incluya un callback de actualización accesible.
        setTimeout(() => {
            const notifEl = document.querySelector('[data-notification-id="pwa-update"]');
            if (!notifEl) return;

            const btn = document.createElement('button');
            btn.textContent = 'Actualizar';
            btn.style.cssText = 'margin-top:8px;padding:6px 16px;border:none;border-radius:4px;background:#228be6;color:white;cursor:pointer;font-size:14px;font-weight:600;';
            btn.addEventListener('click', () => {
                void updateSW(true); // sends SKIP_WAITING + reloads
            });

            const body = notifEl.querySelector('[data-mantine-notification-body]') ??
                         notifEl.querySelector('.mantine-Notification-body');
            if (body) {
                body.appendChild(btn);
            }
        }, 100);
    },
    onOfflineReady() {
        notifications.show({
            title: 'BlendPOS lista offline',
            message: 'La aplicación está lista para funcionar sin conexión.',
            color: 'green',
            autoClose: 5000,
        });
    },
    onRegisteredSW(_swUrl, registration) {
        // Verificar actualizaciones periódicamente (cada 60 min)
        if (registration) {
            setInterval(() => {
                void registration.update();
            }, 60 * 60 * 1000);
        }
    },
});

// ── Background Sync Messages ─────────────────────────────────────────────────
/**
 * Escucha mensajes del Service Worker.
 * Cuando el SW detecta conectividad (Background Sync 'blendpos-sync-ventas'),
 * envía { type: 'SYNC_SALES' } para que el main thread ejecute trySyncQueue().
 */
if ('serviceWorker' in navigator) {
    navigator.serviceWorker.addEventListener('message', (event: MessageEvent<{ type?: string }>) => {
        if (event.data?.type === 'SYNC_SALES') {
            trySyncQueue().catch(console.warn);
        }
    });
}
