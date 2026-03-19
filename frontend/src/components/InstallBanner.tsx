import { Paper, Group, Text, Button, CloseButton } from '@mantine/core';
import { Download } from 'lucide-react';
import { useInstallPrompt } from '../hooks/useInstallPrompt';

/**
 * Banner flotante que invita al usuario a instalar BlendPOS como app.
 *
 * Se muestra solo cuando:
 * - El browser soporta instalación PWA (Chrome, Edge, etc.)
 * - La app no está ya instalada (standalone)
 * - El usuario no descartó el banner en los últimos 7 días
 */
export function InstallBanner() {
    const { canInstall, promptInstall, dismiss } = useInstallPrompt();

    if (!canInstall) return null;

    return (
        <Paper
            shadow="md"
            p="sm"
            radius="md"
            withBorder
            style={{
                position: 'fixed',
                bottom: 16,
                left: '50%',
                transform: 'translateX(-50%)',
                zIndex: 999,
                maxWidth: 440,
                width: 'calc(100% - 32px)',
            }}
        >
            <Group justify="space-between" wrap="nowrap" gap="xs">
                <Group gap="xs" wrap="nowrap" style={{ flex: 1 }}>
                    <Download size={20} style={{ flexShrink: 0 }} />
                    <Text size="sm" fw={500}>
                        Instalá BlendPOS para acceso rápido y offline
                    </Text>
                </Group>
                <Group gap="xs" wrap="nowrap">
                    <Button
                        size="compact-sm"
                        variant="filled"
                        onClick={() => void promptInstall()}
                    >
                        Instalar
                    </Button>
                    <CloseButton size="sm" onClick={dismiss} aria-label="Cerrar" />
                </Group>
            </Group>
        </Paper>
    );
}
