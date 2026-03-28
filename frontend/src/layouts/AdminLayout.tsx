import { useState, useEffect, useCallback } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
    Group, Text, Badge, Avatar, Menu, Burger,
    Tooltip, Divider, Select, Modal, Button, Stack, ThemeIcon,
} from '@mantine/core';
import {
    Package, Boxes, Truck, FileText,
    Users, Calculator, LogOut, ChevronRight, Home,
    BarChart2, Tag, ShoppingBag, Shield, PieChart, Clock, CreditCard, Building2,
    ArrowLeftRight, Warehouse, BrainCircuit, Palette, Lock, Sparkles,
} from 'lucide-react';
import { useAuthStore } from '../store/useAuthStore';
import { useSucursalStore } from '../store/useSucursalStore';
import { useFeature } from '../hooks/useFeature';
import { ThemeToggle } from '../components/ThemeToggle';
import { NAV_FEATURE_MAP, resolvePlanId, getPlanConfig, getMinimumPlanForFeature, type Feature } from '../config/plans';
import styles from './AdminLayout.module.css';

// ── Nav items ────────────────────────────────────────────────────────────────

interface NavItem {
    label: string;
    path: string;
    icon: React.ReactNode;
    roles?: string[];
    /** Feature key required for this nav item. When set and the plan lacks it, show lock. */
    feature?: Feature;
}

const NAV_ITEMS: NavItem[] = [
    { label: 'Dashboard', path: '/admin/dashboard', icon: <BarChart2 size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Productos', path: '/admin/productos', icon: <Package size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Categorias', path: '/admin/categorias', icon: <Tag size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Inventario', path: '/admin/inventario', icon: <Boxes size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Transferencias', path: '/admin/transferencias', icon: <ArrowLeftRight size={18} />, roles: ['admin', 'supervisor'], feature: 'transferencias' },
    { label: 'Stock Sucursal', path: '/admin/stock-sucursal', icon: <Warehouse size={18} />, roles: ['admin', 'supervisor'], feature: 'stock_sucursal' },
    { label: 'Vencimientos', path: '/admin/vencimientos', icon: <Clock size={18} />, roles: ['admin', 'supervisor'], feature: 'vencimientos' },
    { label: 'Proveedores', path: '/admin/proveedores', icon: <Truck size={18} />, roles: ['admin', 'supervisor'], feature: 'proveedores' },
    { label: 'Compras', path: '/admin/compras', icon: <ShoppingBag size={18} />, roles: ['admin', 'supervisor'], feature: 'compras' },
    { label: 'Facturacion', path: '/admin/facturacion', icon: <FileText size={18} />, roles: ['admin', 'supervisor'], feature: 'facturacion_afip' },
    { label: 'Reportes', path: '/admin/reportes', icon: <PieChart size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Asistente IA', path: '/admin/ai', icon: <BrainCircuit size={18} />, roles: ['admin'], feature: 'ai_assistant' },
    { label: 'Clientes / Fiado', path: '/admin/clientes', icon: <CreditCard size={18} />, roles: ['admin', 'supervisor'], feature: 'clientes_management' },
    { label: 'Cierre de Caja', path: '/admin/cierre-caja', icon: <Calculator size={18} /> },
    { label: 'Config. Fiscal', path: '/admin/configuracion-fiscal', icon: <Boxes size={18} />, roles: ['admin'] },
    { label: 'Apariencia POS', path: '/admin/apariencia', icon: <Palette size={18} />, roles: ['admin'], feature: 'apariencia' },
    { label: 'Usuarios', path: '/admin/usuarios', icon: <Users size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Sucursales', path: '/admin/sucursales', icon: <Building2 size={18} />, roles: ['admin'] },
    { label: 'Superadmin', path: '/admin/superadmin', icon: <Shield size={18} />, roles: ['superadmin'] },
];

// ── Rol colors ────────────────────────────────────────────────────────────────

const ROL_COLOR: Record<string, string> = {
    admin: 'red', supervisor: 'yellow', cajero: 'teal', superadmin: 'grape',
};

// ── Plan badge colors ────────────────────────────────────────────────────────

const PLAN_BADGE_COLOR: Record<string, string> = {
    basico: 'gray',
    profesional: 'blue',
    enterprise: 'yellow',
};

// ── NavItemButton (handles feature check per item) ──────────────────────────

function NavItemButton({
    item,
    isActive,
    onClick,
    onLockedClick,
}: {
    item: NavItem;
    isActive: boolean;
    onClick: () => void;
    onLockedClick: (feature: Feature) => void;
}) {
    const featureKey = item.feature;
    const { enabled } = useFeature(featureKey ?? '');
    const isLocked = featureKey != null && !enabled;

    const handleClick = () => {
        if (isLocked && featureKey) {
            onLockedClick(featureKey);
        } else {
            onClick();
        }
    };

    return (
        <button
            className={`${styles.navLink} ${isActive ? styles.navLinkActive : ''} ${isLocked ? styles.navLinkLocked : ''}`}
            onClick={handleClick}
        >
            {item.icon}
            <span style={{ flex: 1 }}>{item.label}</span>
            {isLocked && <Lock size={14} style={{ opacity: 0.5, flexShrink: 0 }} />}
        </button>
    );
}

// ── Component ─────────────────────────────────────────────────────────────────

export function AdminLayout() {
    const [opened, setOpened] = useState(false);
    const [upgradeModal, setUpgradeModal] = useState<{ open: boolean; feature: Feature | null }>({ open: false, feature: null });
    const { user, logout, plan } = useAuthStore();
    const { sucursalId, sucursales, setSucursal, fetchSucursales } = useSucursalStore();
    const navigate = useNavigate();
    const location = useLocation();

    const planId = resolvePlanId(plan ?? 'Basico');
    const planConfig = getPlanConfig(planId);

    useEffect(() => {
        fetchSucursales();
    }, [fetchSucursales]);

    const handleLogout = async () => {
        await logout();
        navigate('/login');
    };

    const handleLockedClick = useCallback((feature: Feature) => {
        setUpgradeModal({ open: true, feature });
    }, []);

    const closeUpgradeModal = useCallback(() => {
        setUpgradeModal({ open: false, feature: null });
    }, []);

    const upgradeFeatureLabel = upgradeModal.feature?.replace(/_/g, ' ') ?? '';
    const upgradePlanName = upgradeModal.feature
        ? getMinimumPlanForFeature(upgradeModal.feature).name
        : '';

    return (
        <div className={styles.shell}>
            <div
                className={`${styles.overlay} ${opened ? styles.overlayOpen : ''}`}
                onClick={() => setOpened(false)}
                aria-hidden
            />

            <aside className={`${styles.sidebar} ${opened ? styles.sidebarOpen : ''}`}>
                <div className={styles.navHeader}>
                    <Text className={styles.brand}>BlendPOS</Text>
                    <Text className={styles.brandSub}>Sistema de Gestion</Text>
                </div>

                <Divider my="xs" />
                <div className={styles.navSectionLabel}>Navegacion</div>

                {NAV_ITEMS
                    .filter((item) => !item.roles || item.roles.includes(user?.rol ?? ''))
                    .map((item) => {
                        const isActive = location.pathname === item.path ||
                            (item.path !== '/' && location.pathname.startsWith(item.path));
                        return (
                            <NavItemButton
                                key={item.path}
                                item={item}
                                isActive={isActive}
                                onClick={() => { navigate(item.path); setOpened(false); }}
                                onLockedClick={handleLockedClick}
                            />
                        );
                    })
                }

                <div style={{ flex: 1 }} />

                {/* ── Subscription Badge ────────────────────────────────── */}
                <div className={styles.navSection}>
                    <div className={styles.planBadgeWrap}>
                        <Badge
                            color={PLAN_BADGE_COLOR[planId] ?? 'gray'}
                            variant="light"
                            size="sm"
                            style={{ width: '100%', textAlign: 'center' }}
                        >
                            Plan {planConfig.name}
                        </Badge>
                    </div>
                    <Tooltip label="Ir al Terminal POS" position="right" withArrow>
                        <button
                            className={styles.navLink}
                            onClick={() => navigate('/')}
                        >
                            <Home size={18} />
                            Volver al POS
                        </button>
                    </Tooltip>
                    <Divider my="md" />
                    <button className={styles.navLinkDanger} onClick={handleLogout}>
                        <LogOut size={18} />
                        Cerrar sesion
                    </button>
                </div>
            </aside>

            <section className={styles.main}>
                <header className={styles.header}>
                    <Group gap="sm" className={styles.headerTitleGroup}>
                        <Burger
                            opened={opened}
                            onClick={() => setOpened((o) => !o)}
                            hiddenFrom="sm"
                            size="sm"
                        />
                        <Text fw={700} size="sm" c="dimmed" className={styles.headerTitle}>
                            BlendPOS — Panel de Administracion
                        </Text>
                    </Group>

                    <div className={styles.userMenu}>
                        {sucursales.length > 0 && (() => {
                            const isCajero = user?.rol === 'cajero';
                            const isLocked = isCajero && !!user?.sucursalId;
                            return (
                                <Select
                                    className={styles.branchSelect}
                                    placeholder="Todas las sucursales"
                                    data={[
                                        ...(!isCajero ? [{ value: '', label: 'Todas las sucursales' }] : []),
                                        ...sucursales.map((s) => ({ value: s.id, label: s.nombre })),
                                    ]}
                                    value={sucursalId ?? ''}
                                    onChange={(val) => {
                                        if (isLocked) return;
                                        const selected = sucursales.find((s) => s.id === val);
                                        setSucursal(val || null, selected?.nombre ?? null);
                                    }}
                                    leftSection={<Building2 size={16} />}
                                    clearable={false}
                                    size="xs"
                                    w={200}
                                    disabled={isLocked}
                                />
                            );
                        })()}
                        <div className={styles.themeToggleWrap}>
                            <ThemeToggle size="sm" />
                        </div>

                        {/* Plan badge in header */}
                        <Badge
                            color={PLAN_BADGE_COLOR[planId] ?? 'gray'}
                            variant="dot"
                            size="sm"
                            className={styles.rolBadge}
                        >
                            {planConfig.name}
                        </Badge>

                        <Badge
                            className={styles.rolBadge}
                            color={ROL_COLOR[user?.rol ?? 'cajero']}
                            variant="light"
                            size="sm"
                        >
                            {user?.rol}
                        </Badge>

                        <Menu shadow="md" width={200} position="bottom-end">
                            <Menu.Target>
                                <Group gap="xs" className={styles.userTrigger}>
                                    <Avatar size="sm" radius="xl" color="blue" className={styles.userAvatar}>
                                        {user?.nombre?.charAt(0)?.toUpperCase() ?? '?'}
                                    </Avatar>
                                    <Text size="sm" fw={500} visibleFrom="sm">
                                        {user?.nombre?.split(' ')[0] ?? ''}
                                    </Text>
                                    <ChevronRight size={14} className={styles.userChevron} />
                                </Group>
                            </Menu.Target>
                            <Menu.Dropdown>
                                <Menu.Label>{user?.email}</Menu.Label>
                                <Menu.Divider />
                                <Menu.Item
                                    color="red"
                                    leftSection={<LogOut size={14} />}
                                    onClick={handleLogout}
                                >
                                    Cerrar sesion
                                </Menu.Item>
                            </Menu.Dropdown>
                        </Menu>
                    </div>
                </header>

                <main className={styles.content}>
                    <Outlet />
                </main>
            </section>

            {/* ── Upgrade Modal ──────────────────────────────────────────── */}
            <Modal
                opened={upgradeModal.open}
                onClose={closeUpgradeModal}
                title="Funcion bloqueada"
                centered
                size="sm"
            >
                <Stack align="center" gap="md" py="md">
                    <ThemeIcon size={56} radius="xl" variant="light" color="blue">
                        <Lock size={28} />
                    </ThemeIcon>
                    <Text fw={600} size="lg" ta="center">
                        Disponible en plan {upgradePlanName}
                    </Text>
                    <Text c="dimmed" size="sm" ta="center">
                        La funcionalidad de <Text span fw={600}>{upgradeFeatureLabel}</Text> no
                        esta incluida en tu plan actual. Mejora tu plan para desbloquear
                        todas las herramientas que tu negocio necesita.
                    </Text>
                    <Button
                        variant="filled"
                        color="blue"
                        leftSection={<Sparkles size={16} />}
                        onClick={closeUpgradeModal}
                        fullWidth
                    >
                        Entendido
                    </Button>
                </Stack>
            </Modal>
        </div>
    );
}
