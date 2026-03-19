// ─────────────────────────────────────────────────────────────────────────────
// AbrirCajaModal — Modal para abrir una sesión de caja antes de operar.
// Mostrado automáticamente si no hay sesión activa en useCajaStore.
// ─────────────────────────────────────────────────────────────────────────────

import { useState } from 'react';
import { Modal, Stack, NumberInput, Button, Text, Alert, Group } from '@mantine/core';
import { TriangleAlert, Store } from 'lucide-react';
import { useCajaStore } from '../../store/useCajaStore';
import { useAuthStore } from '../../store/useAuthStore';

interface Props {
    opened: boolean;
    /** onClose se llama solo cuando la caja se abrió correctamente */
    onSuccess: () => void;
}

export function AbrirCajaModal({ opened, onSuccess }: Props) {
    const { abrir, restaurar, loading, error } = useCajaStore();
    const { user } = useAuthStore();

    // Punto de venta assigned to this user (set in admin panel when creating the user).
    // Falls back to 1 if not assigned.
    const puntoDeVenta = user?.puntoDeVenta ?? 1;
    const [montoInicial, setMontoInicial] = useState<number | string>(0);

    const handleSubmit = async () => {
        const monto = typeof montoInicial === 'number' ? montoInicial : parseFloat(String(montoInicial));
        if (isNaN(monto) || monto < 0) return;

        try {
            await abrir({ punto_de_venta: puntoDeVenta, monto_inicial: monto });
            onSuccess();
        } catch (err) {
            const msg = err instanceof Error ? err.message : '';
            // Si ya existe una caja abierta en ese PDV, recuperar la sesión activa
            if (msg.toLowerCase().includes('ya existe una caja abierta')) {
                await restaurar().catch(() => { });
                const { sesionId } = useCajaStore.getState();
                if (sesionId) onSuccess();
            }
        }
    };

    return (
        <Modal
            opened={opened}
            onClose={() => { /* No permite cerrar sin abrir caja */ }}
            title={
                <Group gap="xs">
                    <Store size={20} />
                    <Text fw={700} size="lg">Abrir Sesión de Caja</Text>
                </Group>
            }
            closeOnClickOutside={false}
            closeOnEscape={false}
            withCloseButton={false}
            size="sm"
            centered
        >
            <Stack gap="md">
                <Text size="sm" c="dimmed">
                    Vas a abrir la caja <Text span fw={700}>#{puntoDeVenta}</Text>. Ingresá el monto en efectivo con el que iniciás.
                </Text>

                {error && (
                    <Alert icon={<TriangleAlert size={16} />} color="red" variant="light">
                        {error}
                    </Alert>
                )}

                <NumberInput
                    label="Monto Inicial en Efectivo ($)"
                    description="Contá los billetes y monedas en la caja"
                    placeholder="0"
                    min={0}
                    decimalScale={2}
                    fixedDecimalScale
                    thousandSeparator="."
                    decimalSeparator=","
                    value={montoInicial}
                    onChange={setMontoInicial}
                    allowNegative={false}
                    size="lg"
                    autoFocus
                />

                <Button
                    fullWidth
                    size="lg"
                    onClick={handleSubmit}
                    loading={loading}
                >
                    Abrir Caja #{puntoDeVenta}
                </Button>
            </Stack>
        </Modal>
    );
}
