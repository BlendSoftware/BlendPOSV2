import { Box, Text, Stack, Group, Badge } from '@mantine/core';
import { ScanBarcode, PackageCheck } from 'lucide-react';
import { useCartStore } from '../../store/useCartStore';
import styles from './LastScannedProduct.module.css';

function formatCurrency(value: number): string {
    return new Intl.NumberFormat('es-AR', {
        style: 'currency',
        currency: 'ARS',
        minimumFractionDigits: 2,
    }).format(value);
}

export function LastScannedProduct() {
    const lastAdded = useCartStore((s) => s.lastAdded);

    if (!lastAdded) {
        return (
            <Box className={styles.emptyContainer}>
                <Stack align="center" gap="xs">
                    <ScanBarcode
                        size={40}
                        strokeWidth={1.2}
                        color="var(--mantine-color-dark-4)"
                    />
                    <Text size="xs" c="dimmed" ta="center" style={{ userSelect: 'none' }}>
                        Último producto escaneado
                    </Text>
                </Stack>
            </Box>
        );
    }

    const unidad = lastAdded.unidadMedida === 'kg'
        ? 'kg'
        : lastAdded.unidadMedida === 'gramo'
            ? 'g'
            : 'ud';
    const effectiveDiscount = Math.max(lastAdded.descuento, lastAdded.promoDescuento ?? 0);

    return (
        <Box className={styles.productContainer}>
            <Group gap="xs" mb={6} justify="space-between" wrap="nowrap">
                <Group gap="xs" wrap="nowrap">
                <PackageCheck size={14} color="#2b8a3e" />
                <Text size="xs" c="green.6" fw={600} tt="uppercase">
                    Producto escaneado
                </Text>
                </Group>
                <Badge size="xs" variant="light" className={styles.unitBadge}>
                    {unidad}
                </Badge>
            </Group>

            <Text className={styles.productName} lineClamp={2}>
                {lastAdded.nombre}
            </Text>

            <Text size="xs" c="dimmed" ff="monospace" mt={4}>
                {lastAdded.codigoBarras}
            </Text>

            {effectiveDiscount > 0 && (
                <Badge mt={8} size="sm" color="orange" variant="light" className={styles.discountBadge}>
                    Descuento aplicado: {effectiveDiscount}%
                </Badge>
            )}

            {lastAdded.promoNombre && (
                <Text size="xs" mt={6} className={styles.promoText}>
                    Promo: {lastAdded.promoNombre}
                </Text>
            )}

            <Group justify="space-between" mt={8} align="flex-end">
                <Stack gap={0}>
                    <Text size="xs" c="dimmed">Precio unit.</Text>
                    <Text size="md" c="dimmed" ff="monospace">
                        {formatCurrency(lastAdded.precio)}
                    </Text>
                </Stack>
                <Stack gap={0} align="flex-end">
                    <Text size="xs" c="dimmed">Cant. / Subtotal</Text>
                    <Text size="lg" fw={800} ff="monospace">
                        {lastAdded.cantidad} × {formatCurrency(lastAdded.subtotal)}
                    </Text>
                </Stack>
            </Group>
        </Box>
    );
}
