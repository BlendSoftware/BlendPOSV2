import { useEffect, useState, memo } from 'react';
import { Group, Text, Badge, Flex, Tooltip, ActionIcon, Modal, Button, useMantineColorScheme } from '@mantine/core';
import { Wifi, WifiOff, User, Printer, PanelLeftOpen, Settings, LogOut, Calculator } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { notifications } from '@mantine/notifications';
import { useAuthStore } from '../../store/useAuthStore';
import { thermalPrinter } from '../../services/ThermalPrinterService';
import { useSyncStatus } from '../../hooks/useSyncStatus';
import { usePrinterStore } from '../../store/usePrinterStore';
import { useCajaStore } from '../../store/useCajaStore';
import { PrinterSettingsModal } from './PrinterSettingsModal';
import { ThemeToggle } from '../ThemeToggle';
import styles from './PosHeader.module.css';

const Clock = memo(function Clock() {
    const [time, setTime] = useState(new Date());

    useEffect(() => {
        setTime(new Date());
        const untilNextMinute = 60_000 - (Date.now() % 60_000);
        let interval: ReturnType<typeof setInterval> | null = null;

        const timeout = setTimeout(() => {
            setTime(new Date());
            interval = setInterval(() => setTime(new Date()), 60_000);
        }, untilNextMinute);

        return () => {
            clearTimeout(timeout);
            if (interval) clearInterval(interval);
        };
    }, []);

    const formattedTime = time.toLocaleTimeString('es-AR', {
        hour: '2-digit', minute: '2-digit', hour12: false,
    });
    const formattedDate = time.toLocaleDateString('es-AR', {
        weekday: 'short', day: '2-digit', month: '2-digit',
    });

    return (
        <div className={styles.clock} aria-label={`Hora actual ${formattedTime}, ${formattedDate}`}>
            <Text className={styles.clockTime} ff="monospace">
                {formattedTime}
            </Text>
            <span className={styles.clockDot} />
            <Text className={styles.clockDate}>
                {formattedDate}
            </Text>
        </div>
    );
});

const ROL_COLOR: Record<string, string> = {
    admin: 'red',
    supervisor: 'yellow',
    cajero: 'teal',
    superadmin: 'grape',
};

export function PosHeader() {
    const [isOnline, setIsOnline] = useState(false);
    const [printerConnected, setPrinterConnected] = useState(thermalPrinter.isConnected);
    const [settingsOpen, setSettingsOpen] = useState(false);
    const [logoutConfirmOpen, setLogoutConfirmOpen] = useState(false);
    const { pending: syncPending, syncState } = useSyncStatus();

    const { user, hasRole, logout } = useAuthStore();
    const { config: printerConfig } = usePrinterStore();
    const { limpiar: limpiarCaja } = useCajaStore();
    const navigate = useNavigate();
    const { colorScheme } = useMantineColorScheme();
    const isDark = colorScheme === 'dark';

    useEffect(() => {
        thermalPrinter.autoConnectIfPossible(printerConfig.baudRate)
            .then((ok) => setPrinterConnected(ok))
            .catch(() => setPrinterConnected(false));
    }, [printerConfig.baudRate]);

    useEffect(() => {
        let mounted = true;
        const BASE_URL = (import.meta.env.VITE_API_BASE as string | undefined) ?? 'http://localhost:8000';

        const checkConnectivity = async () => {
            try {
                const controller = new AbortController();
                const timeout = setTimeout(() => controller.abort(), 5000);
                const res = await fetch(`${BASE_URL}/health`, {
                    method: 'GET',
                    signal: controller.signal,
                    cache: 'no-store',
                });
                clearTimeout(timeout);
                if (mounted) setIsOnline(res.ok);
            } catch {
                if (mounted) setIsOnline(false);
            }
        };

        checkConnectivity();
        const interval = setInterval(checkConnectivity, 10_000);

        const handleOnline = () => checkConnectivity();
        const handleOffline = () => { if (mounted) setIsOnline(false); };
        window.addEventListener('online', handleOnline);
        window.addEventListener('offline', handleOffline);

        return () => {
            mounted = false;
            clearInterval(interval);
            window.removeEventListener('online', handleOnline);
            window.removeEventListener('offline', handleOffline);
        };
    }, []);

    const handlePrinterToggle = async () => {
        if (printerConnected) {
            await thermalPrinter.disconnect();
            setPrinterConnected(false);
            notifications.show({
                title: 'Impresora desconectada',
                message: 'Se cerró la conexión con la impresora térmica.',
                color: 'gray',
                icon: <Printer size={14} />,
            });
            return;
        }

        const ok = await thermalPrinter.connect(printerConfig.baudRate);
        setPrinterConnected(ok);

        if (ok) {
            notifications.show({
                title: 'Impresora conectada',
                message: 'Lista para imprimir tickets ESC/POS.',
                color: 'teal',
                icon: <Printer size={14} />,
            });
            return;
        }

        notifications.show({
            title: 'No se pudo conectar',
            message: 'El navegador no soporta Web Serial o el usuario canceló. Los tickets se mostrarán en consola.',
            color: 'orange',
            icon: <Printer size={14} />,
            autoClose: 5000,
        });
    };

    return (
        <header className={styles.header}>
            <PrinterSettingsModal opened={settingsOpen} onClose={() => setSettingsOpen(false)} />

            <Modal
                opened={logoutConfirmOpen}
                onClose={() => setLogoutConfirmOpen(false)}
                title="¿Cerrar sesión?"
                centered
                size="sm"
            >
                <Text size="sm" c="dimmed" mb="lg">
                    ¿Estás seguro que querés cerrar sesión?
                </Text>
                <Group justify="flex-end" gap="sm">
                    <Button variant="default" autoFocus onClick={() => setLogoutConfirmOpen(false)}>
                        Cancelar
                    </Button>
                    <Button
                        color="red"
                        onClick={async () => {
                            setLogoutConfirmOpen(false);
                            limpiarCaja();
                            await logout();
                            navigate('/login');
                        }}
                    >
                        Sí, cerrar sesión
                    </Button>
                </Group>
            </Modal>

            <div className={styles.headerInner}>
                <div className={styles.mainRow}>
                    <div className={styles.brandBlock}>
                        <Text className={styles.brand}>BlendPOS</Text>
                        <Text className={styles.brandSub}>Punto de Venta</Text>
                    </div>

                    <div className={styles.operatorBlock}>
                        <User size={16} color="#909296" />
                        <Text size="xs" c="dimmed">Operador</Text>
                        <Text size="sm" fw={700} c={isDark ? 'white' : 'dark.7'}>
                            {user?.nombre ?? 'Cargándose…'}
                        </Text>
                        {user?.rol && (
                            <Badge color={ROL_COLOR[user.rol] ?? 'gray'} size="xs" variant="light">
                                {user.rol}
                            </Badge>
                        )}
                        <Text size="xs" c="dimmed">
                            Terminal #{user?.puntoDeVenta != null
                                ? String(user.puntoDeVenta).padStart(2, '0')
                                : 'POS'}
                        </Text>
                    </div>

                    <Group gap="xs" className={styles.actionsBlock}>
                        <Tooltip label={printerConnected ? 'Desconectar impresora' : 'Conectar impresora térmica'} withArrow>
                            <ActionIcon
                                variant={printerConnected ? 'filled' : 'subtle'}
                                color={printerConnected ? 'teal' : 'gray'}
                                size="md"
                                onClick={handlePrinterToggle}
                            >
                                <Printer size={16} />
                            </ActionIcon>
                        </Tooltip>

                        <Tooltip label="Configuración de impresora" withArrow>
                            <ActionIcon
                                variant="subtle"
                                color="gray"
                                size="md"
                                onClick={() => setSettingsOpen(true)}
                            >
                                <Settings size={16} />
                            </ActionIcon>
                        </Tooltip>

                        <Tooltip label="Cerrar Caja" withArrow>
                            <ActionIcon
                                variant="subtle"
                                color="orange"
                                size="md"
                                onClick={() => navigate('/admin/cierre-caja')}
                            >
                                <Calculator size={16} />
                            </ActionIcon>
                        </Tooltip>

                        {hasRole(['admin', 'supervisor']) && (
                            <Tooltip label="Panel Admin" withArrow>
                                <ActionIcon
                                    variant="subtle"
                                    color="blue"
                                    size="md"
                                    onClick={() => navigate('/admin/dashboard')}
                                >
                                    <PanelLeftOpen size={16} />
                                </ActionIcon>
                            </Tooltip>
                        )}

                        <ThemeToggle size="md" />

                        <Tooltip label="Cerrar sesión" withArrow>
                            <ActionIcon
                                variant="light"
                                color="red"
                                size="md"
                                onClick={() => setLogoutConfirmOpen(true)}
                            >
                                <LogOut size={16} />
                            </ActionIcon>
                        </Tooltip>
                    </Group>
                </div>

                <Flex align="center" justify="space-between" className={styles.statusRow}>
                    {/* Unified sync + connectivity badge (F1-7) */}
                    {(() => {
                        // Red: offline
                        if (!isOnline) return (
                            <Tooltip label="Sin conexión al servidor" withArrow>
                                <Badge size="lg" variant="light" className={`${styles.statusBadge} ${styles.statusOffline}`}>
                                    <Group gap={4}>
                                        <span className={`${styles.statusDot} ${styles.statusDotOffline}`} />
                                        <WifiOff size={12} />
                                        <span className={styles.statusText}>Sin conexión</span>
                                    </Group>
                                </Badge>
                            </Tooltip>
                        );
                        // Yellow: online but pending sales or actively syncing
                        if (syncPending > 0 || syncState === 'syncing') return (
                            <Tooltip
                                label={syncState === 'syncing'
                                    ? 'Sincronizando con el servidor…'
                                    : `${syncPending} venta${syncPending !== 1 ? 's' : ''} pendiente${syncPending !== 1 ? 's' : ''} de sincronizar`}
                                withArrow
                            >
                                <Badge size="lg" variant="light" className={`${styles.statusBadge} ${styles.statusSync}`}>
                                    <Group gap={4}>
                                        <span className={`${styles.statusDot} ${styles.statusDotSync}`} />
                                        <Wifi size={12} />
                                        <span className={styles.statusText}>{syncState === 'syncing' ? `⟳ ${syncPending}` : `Sync ${syncPending}`}</span>
                                    </Group>
                                </Badge>
                            </Tooltip>
                        );
                        // Green: online + fully synced
                        return (
                            <Tooltip label="Conectado y sincronizado" withArrow>
                                <Badge size="lg" variant="light" className={`${styles.statusBadge} ${styles.statusOnline}`}>
                                    <Group gap={4}>
                                        <span className={`${styles.statusDot} ${styles.statusDotOnline}`} />
                                        <Wifi size={12} />
                                        <span className={styles.statusText}>Conectado</span>
                                    </Group>
                                </Badge>
                            </Tooltip>
                        );
                    })()}
                    <Clock />
                </Flex>
            </div>
        </header>
    );
}
