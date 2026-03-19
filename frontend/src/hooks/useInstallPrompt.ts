import { useState, useEffect, useCallback } from 'react';

/**
 * Interfaz del evento beforeinstallprompt (no está en los tipos estándar de TS).
 */
interface BeforeInstallPromptEvent extends Event {
    readonly platforms: string[];
    readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
    prompt(): Promise<void>;
}

const DISMISS_KEY = 'blendpos-install-dismissed-at';
const DISMISS_DAYS = 7;

function isDismissedRecently(): boolean {
    const raw = localStorage.getItem(DISMISS_KEY);
    if (!raw) return false;
    const dismissedAt = Number(raw);
    const daysSince = (Date.now() - dismissedAt) / (1000 * 60 * 60 * 24);
    return daysSince < DISMISS_DAYS;
}

/**
 * Hook para manejar el prompt de instalación PWA.
 *
 * - `canInstall`: true cuando el browser ofrece instalación Y el usuario no
 *   descartó el prompt en los últimos 7 días.
 * - `promptInstall()`: muestra el prompt nativo del browser.
 * - `dismiss()`: oculta el banner sin instalar (recordar por 7 días).
 */
export function useInstallPrompt() {
    const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null);
    const [canInstall, setCanInstall] = useState(false);

    useEffect(() => {
        // Si ya se descartó recientemente, no escuchar el evento
        if (isDismissedRecently()) return;

        // Si ya está instalada como standalone, no mostrar
        if (window.matchMedia('(display-mode: standalone)').matches) return;

        const handler = (e: Event) => {
            e.preventDefault();
            setDeferredPrompt(e as BeforeInstallPromptEvent);
            setCanInstall(true);
        };

        window.addEventListener('beforeinstallprompt', handler);

        return () => {
            window.removeEventListener('beforeinstallprompt', handler);
        };
    }, []);

    const promptInstall = useCallback(async () => {
        if (!deferredPrompt) return;

        await deferredPrompt.prompt();
        const { outcome } = await deferredPrompt.userChoice;

        if (outcome === 'dismissed') {
            localStorage.setItem(DISMISS_KEY, String(Date.now()));
        }

        // Después de prompt (aceptado o no), limpiar
        setDeferredPrompt(null);
        setCanInstall(false);
    }, [deferredPrompt]);

    const dismiss = useCallback(() => {
        localStorage.setItem(DISMISS_KEY, String(Date.now()));
        setDeferredPrompt(null);
        setCanInstall(false);
    }, []);

    return { canInstall, promptInstall, dismiss } as const;
}
