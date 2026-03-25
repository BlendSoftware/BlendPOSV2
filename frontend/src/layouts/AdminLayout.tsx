import { useState, useEffect } from 'react';
import { Outlet, useNavigate, NavLink } from 'react-router-dom';
import {
    Group, Text, Badge, Avatar, Menu, Burger,
    Tooltip, Divider, Select,
} from '@mantine/core';
import {
    Package, Boxes, Truck, FileText,
    Users, Calculator, LogOut, ChevronRight, Home,
    BarChart2, Tag, ShoppingBag, Shield, PieChart, Clock, CreditCard, Building2,
    ArrowLeftRight, Warehouse, BrainCircuit,
} from 'lucide-react';
import { useAuthStore } from '../store/useAuthStore';
import { useSucursalStore } from '../store/useSucursalStore';
import { getAlertasStock } from '../services/api/inventario';
import { ThemeToggle } from '../components/ThemeToggle';
import styles from './AdminLayout.module.css';

// ── Nav items ────────────────────────────────────────────────────────────────

interface NavItem {
    label: string;
    path: string;
    icon: React.ReactNode;
    roles?: string[];
}

const NAV_ITEMS: NavItem[] = [
    { label: 'Dashboard', path: '/admin/dashboard', icon: <BarChart2 size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Productos', path: '/admin/productos', icon: <Package size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Categorías', path: '/admin/categorias', icon: <Tag size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Inventario', path: '/admin/inventario', icon: <Boxes size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Transferencias', path: '/admin/transferencias', icon: <ArrowLeftRight size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Stock Sucursal', path: '/admin/stock-sucursal', icon: <Warehouse size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Vencimientos', path: '/admin/vencimientos', icon: <Clock size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Proveedores', path: '/admin/proveedores', icon: <Truck size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Compras', path: '/admin/compras', icon: <ShoppingBag size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Facturación', path: '/admin/facturacion', icon: <FileText size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Reportes', path: '/admin/reportes', icon: <PieChart size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Asistente IA', path: '/admin/ai', icon: <BrainCircuit size={18} />, roles: ['admin'] },
    { label: 'Clientes / Fiado', path: '/admin/clientes', icon: <CreditCard size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Cierre de Caja', path: '/admin/cierre-caja', icon: <Calculator size={18} /> },
    { label: 'Config. Fiscal', path: '/admin/configuracion-fiscal', icon: <Boxes size={18} />, roles: ['admin'] },
    { label: 'Usuarios', path: '/admin/usuarios', icon: <Users size={18} />, roles: ['admin', 'supervisor'] },
    { label: 'Sucursales', path: '/admin/sucursales', icon: <Building2 size={18} />, roles: ['admin'] },
    { label: 'Superadmin', path: '/admin/superadmin', icon: <Shield size={18} />, roles: ['superadmin'] },
];

// ── Rol colors ────────────────────────────────────────────────────────────────

const ROL_COLOR: Record<string, string> = {
    admin: 'red', supervisor: 'yellow', cajero: 'teal', superadmin: 'grape',
};

// ── Component ─────────────────────────────────────────────────────────────────

export function AdminLayout() {
    const [opened, setOpened] = useState(false);
    const { user, logout } = useAuthStore();
    const { sucursalId, sucursales, setSucursal, fetchSucursales } = useSucursalStore();
    const navigate = useNavigate();
    const [stockAlertCount, setStockAlertCount] = useState(0);

    useEffect(() => {
        fetchSucursales();
    }, [fetchSucursales]);

    useEffect(() => {
        getAlertasStock().then((a) => setStockAlertCount(a.length)).catch(() => {});
    }, []);

    const handleLogout = async () => {
        await logout();
        navigate('/login');
    };

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
                    <Text className={styles.brandSub}>Sistema de Gestión</Text>
                </div>

                <Divider my="xs" />
                <div className={styles.navSectionLabel}>Navegación</div>

                {NAV_ITEMS
                    .filter((item) => !item.roles || item.roles.includes(user?.rol ?? ''))
                    .map((item) => (
                            <NavLink
                                key={item.path}
                                to={item.path}
                                className={({ isActive }) => `${styles.navLink} ${isActive ? styles.navLinkActive : ''}`}
                                onClick={() => setOpened(false)}
                                end={item.path === '/admin/dashboard'}
                            >
                                {item.icon}
                                {item.label}
                                {item.path === '/admin/inventario' && stockAlertCount > 0 && (
                                    <Badge size="xs" circle color="red" ml="auto">{stockAlertCount}</Badge>
                                )}
                            </NavLink>
                        ))
                }

                <div style={{ flex: 1 }} />
                <div className={styles.navSection}>
                    <Tooltip label="Ir al Terminal POS" position="right" withArrow>
                        <NavLink
                            to="/"
                            className={styles.navLink}
                        >
                            <Home size={18} />
                            Volver al POS
                        </NavLink>
                    </Tooltip>
                    <Divider my="md" />
                    <button className={styles.navLinkDanger} onClick={handleLogout}>
                        <LogOut size={18} />
                        Cerrar sesión
                    </button>
                </div>
            </aside>

            <section className={styles.main}>
                <header className={styles.header}>
                    <Group gap="sm">
                        <Burger
                            opened={opened}
                            onClick={() => setOpened((o) => !o)}
                            hiddenFrom="sm"
                            size="sm"
                        />
                        <Text fw={700} size="sm" c="dimmed">
                            BlendPOS — Panel de Administración
                        </Text>
                    </Group>

                    <div className={styles.userMenu}>
                        {sucursales.length > 0 && (
                            <Select
                                placeholder="Todas las sucursales"
                                data={[
                                    { value: '', label: 'Todas las sucursales' },
                                    ...sucursales.map((s) => ({ value: s.id, label: s.nombre })),
                                ]}
                                value={sucursalId ?? ''}
                                onChange={(val) => {
                                    const selected = sucursales.find((s) => s.id === val);
                                    setSucursal(val || null, selected?.nombre ?? null);
                                }}
                                leftSection={<Building2 size={16} />}
                                clearable={false}
                                size="xs"
                                w={200}
                                styles={{ input: { fontWeight: 500 } }}
                            />
                        )}
                        <ThemeToggle size="sm" />

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
                                <Group gap="xs" style={{ cursor: 'pointer' }}>
                                    <Avatar size="sm" radius="xl" color="blue">
                                        {user?.nombre.charAt(0).toUpperCase()}
                                    </Avatar>
                                    <Text size="sm" fw={500} visibleFrom="sm">
                                        {user?.nombre.split(' ')[0]}
                                    </Text>
                                    <ChevronRight size={14} />
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
                                    Cerrar sesión
                                </Menu.Item>
                            </Menu.Dropdown>
                        </Menu>
                    </div>
                </header>

                <main className={styles.content}>
                    <Outlet />
                </main>
            </section>
        </div>
    );
}
