import { Stack, Text, Button, Divider, Group, Badge, Box } from '@mantine/core';
import { CreditCard, X, Percent } from 'lucide-react';
import { useCartStore } from '../../store/useCartStore';
import { usePOSUIStore } from '../../store/usePOSUIStore';
import { LastScannedProduct } from './LastScannedProduct';
import styles from './TotalPanel.module.css';
import { formatCurrency } from '../../utils/format';

export function TotalPanel() {
    const total = useCartStore((s) => s.total);
    const descuentoGlobal = useCartStore((s) => s.descuentoGlobal);
    const totalConDescuento = useCartStore((s) => s.totalConDescuento);
    const cart = useCartStore((s) => s.cart);
    const openPaymentModal = usePOSUIStore((s) => s.openPaymentModal);
    const clearCart = useCartStore((s) => s.clearCart);
    const openDiscountModal = usePOSUIStore((s) => s.openDiscountModal);

    const itemCount = cart.reduce((sum, item) => sum + item.cantidad, 0);
    const hasDiscount = descuentoGlobal > 0;
    const displayTotal = hasDiscount ? totalConDescuento : total;

    return (
        <div className={styles.panel}>
            <Stack gap="xs" className={styles.scanSection}>
                <Text size="xs" c="dimmed" tt="uppercase" fw={700} className={styles.blockTitle}>
                    Último Escaneado
                </Text>
                <LastScannedProduct />
            </Stack>

            <Divider my="sm" />

            <Group grow className={styles.metricsRow}>
                <Box className={styles.metricCard}>
                    <Text size="xs" c="dimmed">Artículos</Text>
                    <Text size="lg" fw={700} ff="monospace">{itemCount}</Text>
                </Box>
                <Box className={styles.metricCard}>
                    <Text size="xs" c="dimmed">Descuento</Text>
                    <Text size="lg" fw={700} ff="monospace">{hasDiscount ? `${descuentoGlobal}%` : '0%'}</Text>
                </Box>
            </Group>

            <Stack gap="xs" align="center" className={styles.totalSection}>
                <Text size="sm" c="dimmed" tt="uppercase" fw={700} className={styles.blockTitle}>
                    Total a Cobrar
                </Text>

                {hasDiscount && (
                    <Box className={styles.originalPrice}>
                        <Text size="md" c="dimmed" td="line-through" ff="monospace">
                            {formatCurrency(total)}
                        </Text>
                        <Badge color="orange" variant="light" size="sm">
                            -{descuentoGlobal}%
                        </Badge>
                    </Box>
                )}

                <Text className={`${styles.totalAmount} ${hasDiscount ? styles.totalDiscount : styles.totalNormal}`} fw={800}>
                    {formatCurrency(displayTotal)}
                </Text>
            </Stack>

            <Divider my="md" />

            <Stack gap="sm" className={styles.actions}>
                <Button
                    size="xl"
                    color="green"
                    leftSection={<CreditCard size={22} />}
                    fullWidth
                    onClick={openPaymentModal}
                    disabled={cart.length === 0}
                    className={`${styles.actionButton} ${styles.payButton}`}
                >
                    <Stack gap={0} align="flex-start">
                        <Text size="lg" fw={700}>COBRAR</Text>
                        <Text size="xs" className={styles.shortcutLabel}>F10</Text>
                    </Stack>
                </Button>

                <Button
                    size="md"
                    color="blue"
                    variant="light"
                    leftSection={<Percent size={18} />}
                    fullWidth
                    onClick={openDiscountModal}
                    disabled={cart.length === 0}
                    className={`${styles.actionButton} ${styles.discountButton}`}
                >
                    <Stack gap={0} align="flex-start">
                        <Text size="sm" fw={700}>
                            {hasDiscount ? `Descuento (${descuentoGlobal}%)` : 'DESCUENTO'}
                        </Text>
                        <Text size="xs" className={styles.shortcutLabel}>F8</Text>
                    </Stack>
                </Button>

                <Button
                    size="xl"
                    color="red"
                    variant="outline"
                    leftSection={<X size={22} />}
                    fullWidth
                    onClick={() => {
                        if (cart.length > 0 && !window.confirm('¿Cancelar la venta y vaciar el carrito?')) return;
                        clearCart();
                    }}
                    disabled={cart.length === 0}
                    className={`${styles.actionButton} ${styles.cancelButton}`}
                >
                    <Stack gap={0} align="flex-start">
                        <Text size="lg" fw={700}>CANCELAR</Text>
                    </Stack>
                </Button>
            </Stack>
        </div>
    );
}
