// ─────────────────────────────────────────────────────────────────────────────
// ReportesPage — Analytics dashboard con date range picker y KPIs
// Usa SOLO componentes Mantine — sin chart libraries externas.
// ─────────────────────────────────────────────────────────────────────────────

import { useCallback, useEffect, useRef, useState } from 'react';
import {
    Alert,
    Badge,
    Group,
    Paper,
    Progress,
    SegmentedControl,
    SimpleGrid,
    Skeleton,
    Stack,
    Table,
    Tabs,
    Text,
    ThemeIcon,
    Title,
    Tooltip,
    ActionIcon,
} from '@mantine/core';
import { DatePickerInput, type DatesRangeValue } from '@mantine/dates';
import {
    AlertTriangle,
    Calendar,
    Clock,
    CreditCard,
    DollarSign,
    Package,
    RefreshCw,
    ShoppingCart,
    TrendingUp,
    User,
} from 'lucide-react';
import { formatARS } from '../../utils/format';
import {
    getResumen,
    getTopProductos,
    getMediosPago,
    getVentasPorPeriodo,
    getCajeros,
    getTurnos,
    type ResumenResponse,
    type TopProductoResponse,
    type MedioPagoResponse,
    type VentaPeriodoResponse,
    type CajeroResponse,
    type TurnoResponse,
    type Agrupacion,
} from '../../services/api/reportes';

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Format Date to YYYY-MM-DD for API params */
function toDateStr(d: Date): string {
    return d.toLocaleDateString('en-CA');
}

function defaultRange(): [Date, Date] {
    const hasta = new Date();
    const desde = new Date();
    desde.setDate(hasta.getDate() - 30);
    return [desde, hasta];
}

const MEDIO_PAGO_COLOR: Record<string, string> = {
    efectivo: 'teal',
    debito: 'blue',
    credito: 'violet',
    transferencia: 'cyan',
    qr: 'orange',
};

// ── KPI Card ─────────────────────────────────────────────────────────────────

function KpiCard({
    label,
    value,
    sub,
    icon,
    color,
    loading,
}: {
    label: string;
    value: string;
    sub: string;
    icon: React.ReactNode;
    color: string;
    loading: boolean;
}) {
    if (loading) return <Skeleton h={110} radius="md" />;

    return (
        <Paper
            p="lg"
            radius="md"
            withBorder
            style={{ borderLeft: `4px solid var(--mantine-color-${color}-5)` }}
        >
            <Group justify="space-between" align="flex-start" wrap="nowrap">
                <Stack gap={4}>
                    <Text size="xs" c="dimmed" tt="uppercase" fw={700} style={{ letterSpacing: '0.08em' }}>
                        {label}
                    </Text>
                    <Title order={2} fw={900} lh={1.1} c={`${color}.4`}>
                        {value}
                    </Title>
                    <Text size="xs" c="dimmed">
                        {sub}
                    </Text>
                </Stack>
                <ThemeIcon
                    variant="gradient"
                    gradient={{ from: `${color}.9`, to: `${color}.5`, deg: 135 }}
                    size={46}
                    radius="md"
                >
                    {icon}
                </ThemeIcon>
            </Group>
        </Paper>
    );
}

// ── Simple Bar Chart (Mantine Progress-based) ────────────────────────────────

function SimpleBarChart({
    data,
    loading,
}: {
    data: VentaPeriodoResponse[];
    loading: boolean;
}) {
    if (loading) return <Skeleton h={280} radius="md" />;

    if (data.length === 0) {
        return (
            <Text size="sm" c="dimmed" ta="center" py="xl">
                Sin datos para el periodo seleccionado
            </Text>
        );
    }

    const maxTotal = Math.max(...data.map((d) => d.total), 1);

    return (
        <Stack gap="xs">
            {data.map((item, idx) => {
                const pct = (item.total / maxTotal) * 100;
                return (
                    <Group key={idx} gap="sm" wrap="nowrap" align="center">
                        <Text
                            size="xs"
                            c="dimmed"
                            fw={600}
                            style={{ minWidth: 80, textAlign: 'right', flexShrink: 0 }}
                        >
                            {item.periodo}
                        </Text>
                        <Tooltip
                            label={`${formatARS(item.total)} — ${item.cantidad} ventas`}
                            withArrow
                        >
                            <div style={{ flex: 1, position: 'relative' }}>
                                <Progress
                                    value={pct}
                                    color="blue"
                                    size="xl"
                                    radius="sm"
                                    aria-label={`${item.periodo}: ${formatARS(item.total)}`}
                                />
                            </div>
                        </Tooltip>
                        <Text
                            size="xs"
                            fw={700}
                            style={{ minWidth: 90, textAlign: 'right', flexShrink: 0 }}
                        >
                            {formatARS(item.total)}
                        </Text>
                    </Group>
                );
            })}
        </Stack>
    );
}

// ── Main Page ────────────────────────────────────────────────────────────────

export function ReportesPage() {
    const [defaultDesde, defaultHasta] = defaultRange();
    const [dateRange, setDateRange] = useState<DatesRangeValue>([defaultDesde, defaultHasta]);
    const [agrupacion, setAgrupacion] = useState<Agrupacion>('dia');

    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const [activeTab, setActiveTab] = useState<string | null>('general');

    const [resumen, setResumen] = useState<ResumenResponse | null>(null);
    const [topProductos, setTopProductos] = useState<TopProductoResponse[]>([]);
    const [mediosPago, setMediosPago] = useState<MedioPagoResponse[]>([]);
    const [ventasPeriodo, setVentasPeriodo] = useState<VentaPeriodoResponse[]>([]);
    const [cajeros, setCajeros] = useState<CajeroResponse[]>([]);
    const [turnos, setTurnos] = useState<TurnoResponse[]>([]);

    const fetchData = useCallback(
        async (showRefresh = false) => {
            const desde = dateRange[0];
            const hasta = dateRange[1];
            if (!desde || !hasta) return;

            if (showRefresh) setRefreshing(true);
            setError(null);

            try {
                const desdeStr = toDateStr(desde instanceof Date ? desde : new Date(desde));
                const hastaStr = toDateStr(hasta instanceof Date ? hasta : new Date(hasta));

                const [resumenRes, topRes, mediosRes, periodoRes, cajerosRes, turnosRes] = await Promise.allSettled([
                    getResumen(desdeStr, hastaStr),
                    getTopProductos(desdeStr, hastaStr, 10),
                    getMediosPago(desdeStr, hastaStr),
                    getVentasPorPeriodo(desdeStr, hastaStr, agrupacion),
                    getCajeros(desdeStr, hastaStr),
                    getTurnos(desdeStr, hastaStr),
                ]);

                if (resumenRes.status === 'fulfilled') setResumen(resumenRes.value);
                else setResumen(null);

                if (topRes.status === 'fulfilled') setTopProductos(topRes.value);
                else setTopProductos([]);

                if (mediosRes.status === 'fulfilled') setMediosPago(mediosRes.value);
                else setMediosPago([]);

                if (periodoRes.status === 'fulfilled') setVentasPeriodo(periodoRes.value);
                else setVentasPeriodo([]);

                if (cajerosRes.status === 'fulfilled') setCajeros(cajerosRes.value);
                else setCajeros([]);

                if (turnosRes.status === 'fulfilled') setTurnos(turnosRes.value);
                else setTurnos([]);

                // If ALL failed, show error
                const allFailed = [resumenRes, topRes, mediosRes, periodoRes, cajerosRes, turnosRes].every(
                    (r) => r.status === 'rejected',
                );
                if (allFailed) {
                    const reason =
                        resumenRes.status === 'rejected'
                            ? String(resumenRes.reason)
                            : 'Error desconocido';
                    setError(reason);
                }
            } catch (err) {
                setError(String(err));
            } finally {
                setLoading(false);
                setRefreshing(false);
            }
        },
        [dateRange, agrupacion],
    );

    const fetchRef = useRef(fetchData);
    fetchRef.current = fetchData;

    // Fetch on mount and when dateRange/agrupacion changes
    useEffect(() => {
        if (!dateRange[0] || !dateRange[1]) return;
        setLoading(true);
        fetchRef.current();
    }, [dateRange, agrupacion]);

    // ── Derived values ───────────────────────────────────────────────────────

    const totalMediosPago = mediosPago.reduce((s, m) => s + m.total, 0);

    return (
        <Stack gap="xl">
            {/* ── Header + Date Picker ────────────────────────────────────────── */}
            <Group justify="space-between" align="flex-end" wrap="wrap">
                <div>
                    <Title order={2} fw={800} c="blue.4">
                        Reportes
                    </Title>
                    <Text c="dimmed" size="sm">
                        Analisis de ventas y rendimiento del negocio
                    </Text>
                </div>
                <Group gap="md" align="flex-end">
                    <DatePickerInput
                        type="range"
                        label="Periodo"
                        placeholder="Seleccionar rango"
                        value={dateRange}
                        onChange={setDateRange}
                        maxDate={new Date()}
                        leftSection={<Calendar size={16} />}
                        clearable={false}
                        size="sm"
                        w={280}
                    />
                    <Tooltip label="Actualizar datos">
                        <ActionIcon
                            variant="subtle"
                            color="gray"
                            size="lg"
                            onClick={() => fetchData(true)}
                            loading={refreshing}
                            aria-label="Actualizar reportes"
                        >
                            <RefreshCw size={18} />
                        </ActionIcon>
                    </Tooltip>
                </Group>
            </Group>

            {/* ── Error Alert ─────────────────────────────────────────────────── */}
            {error && (
                <Alert
                    icon={<AlertTriangle size={18} />}
                    color="red"
                    title="Error al cargar reportes"
                    withCloseButton
                    onClose={() => setError(null)}
                >
                    {error}
                </Alert>
            )}

            {/* ── Tabs ─────────────────────────────────────────────────────── */}
            <Tabs value={activeTab} onChange={setActiveTab}>
                <Tabs.List>
                    <Tabs.Tab value="general" leftSection={<TrendingUp size={16} />}>
                        General
                    </Tabs.Tab>
                    <Tabs.Tab value="cajeros" leftSection={<User size={16} />}>
                        Por Cajero
                    </Tabs.Tab>
                    <Tabs.Tab value="turnos" leftSection={<Clock size={16} />}>
                        Turnos
                    </Tabs.Tab>
                </Tabs.List>

                {/* ── General Tab ─────────────────────────────────────────────── */}
                <Tabs.Panel value="general" pt="lg">
                    <Stack gap="xl">
                        {/* KPI Cards */}
                        <SimpleGrid cols={{ base: 1, sm: 3 }}>
                            <KpiCard
                                label="Total ventas"
                                value={formatARS(resumen?.total_ventas ?? 0)}
                                sub="Facturación del periodo"
                                icon={<DollarSign size={22} />}
                                color="teal"
                                loading={loading}
                            />
                            <KpiCard
                                label="Cantidad de ventas"
                                value={String(resumen?.cantidad_ventas ?? 0)}
                                sub="Transacciones completadas"
                                icon={<ShoppingCart size={22} />}
                                color="blue"
                                loading={loading}
                            />
                            <KpiCard
                                label="Ticket promedio"
                                value={formatARS(resumen?.ticket_promedio ?? 0)}
                                sub="Promedio por transacción"
                                icon={<TrendingUp size={22} />}
                                color="violet"
                                loading={loading}
                            />
                        </SimpleGrid>

                        {/* Ventas por periodo (bar chart) */}
                        <Paper p="lg" radius="md" withBorder>
                            <Group justify="space-between" mb="lg">
                                <div>
                                    <Title order={5} c="blue.4">
                                        Ventas por periodo
                                    </Title>
                                    <Text size="xs" c="dimmed">
                                        Evolución de ventas en el rango seleccionado
                                    </Text>
                                </div>
                                <SegmentedControl
                                    value={agrupacion}
                                    onChange={(v) => setAgrupacion(v as Agrupacion)}
                                    data={[
                                        { value: 'dia', label: 'Día' },
                                        { value: 'semana', label: 'Semana' },
                                        { value: 'mes', label: 'Mes' },
                                    ]}
                                    size="xs"
                                />
                            </Group>
                            <SimpleBarChart data={ventasPeriodo} loading={loading} />
                        </Paper>

                        {/* Tables Row */}
                        <SimpleGrid cols={{ base: 1, md: 2 }}>
                            {/* Top 10 productos */}
                            <Paper p="lg" radius="md" withBorder>
                                <Group justify="space-between" mb="md">
                                    <div>
                                        <Title order={5} c="violet.4">
                                            Top 10 productos
                                        </Title>
                                        <Text size="xs" c="dimmed">
                                            Por recaudación en el periodo
                                        </Text>
                                    </div>
                                    <ThemeIcon variant="light" color="violet" size="md" radius="sm">
                                        <Package size={16} />
                                    </ThemeIcon>
                                </Group>

                                {loading ? (
                                    <Stack gap="xs">
                                        {Array.from({ length: 5 }, (_, i) => (
                                            <Skeleton key={i} h={36} radius="sm" />
                                        ))}
                                    </Stack>
                                ) : topProductos.length === 0 ? (
                                    <Text size="sm" c="dimmed" ta="center" py="xl">
                                        Sin productos vendidos en el periodo
                                    </Text>
                                ) : (
                                    <Table verticalSpacing="sm" highlightOnHover withRowBorders={false}>
                                        <Table.Thead>
                                            <Table.Tr
                                                style={{
                                                    borderBottom: '1px solid var(--mantine-color-default-border)',
                                                }}
                                            >
                                                <Table.Th>
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        #
                                                    </Text>
                                                </Table.Th>
                                                <Table.Th>
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        Producto
                                                    </Text>
                                                </Table.Th>
                                                <Table.Th ta="right">
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        Cantidad
                                                    </Text>
                                                </Table.Th>
                                                <Table.Th ta="right">
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        Total
                                                    </Text>
                                                </Table.Th>
                                            </Table.Tr>
                                        </Table.Thead>
                                        <Table.Tbody>
                                            {topProductos.map((p, idx) => (
                                                <Table.Tr key={idx}>
                                                    <Table.Td>
                                                        <Badge
                                                            variant="light"
                                                            color="violet"
                                                            size="sm"
                                                            circle
                                                        >
                                                            {idx + 1}
                                                        </Badge>
                                                    </Table.Td>
                                                    <Table.Td>
                                                        <Text size="sm" fw={500} lineClamp={1}>
                                                            {p.nombre}
                                                        </Text>
                                                    </Table.Td>
                                                    <Table.Td ta="right">
                                                        <Text size="sm" c="dimmed">
                                                            {p.cantidad_vendida}
                                                        </Text>
                                                    </Table.Td>
                                                    <Table.Td ta="right">
                                                        <Text size="sm" fw={700}>
                                                            {formatARS(p.total_recaudado)}
                                                        </Text>
                                                    </Table.Td>
                                                </Table.Tr>
                                            ))}
                                        </Table.Tbody>
                                    </Table>
                                )}
                            </Paper>

                            {/* Medios de pago */}
                            <Paper p="lg" radius="md" withBorder>
                                <Group justify="space-between" mb="md">
                                    <div>
                                        <Title order={5} c="blue.4">
                                            Medios de pago
                                        </Title>
                                        <Text size="xs" c="dimmed">
                                            Distribución por método de pago
                                        </Text>
                                    </div>
                                    <ThemeIcon variant="light" color="blue" size="md" radius="sm">
                                        <CreditCard size={16} />
                                    </ThemeIcon>
                                </Group>

                                {loading ? (
                                    <Stack gap="xs">
                                        {Array.from({ length: 4 }, (_, i) => (
                                            <Skeleton key={i} h={36} radius="sm" />
                                        ))}
                                    </Stack>
                                ) : mediosPago.length === 0 ? (
                                    <Text size="sm" c="dimmed" ta="center" py="xl">
                                        Sin datos de medios de pago
                                    </Text>
                                ) : (
                                    <Table verticalSpacing="sm" highlightOnHover withRowBorders={false}>
                                        <Table.Thead>
                                            <Table.Tr
                                                style={{
                                                    borderBottom: '1px solid var(--mantine-color-default-border)',
                                                }}
                                            >
                                                <Table.Th>
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        Método
                                                    </Text>
                                                </Table.Th>
                                                <Table.Th ta="right">
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        Ventas
                                                    </Text>
                                                </Table.Th>
                                                <Table.Th ta="right">
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        Total
                                                    </Text>
                                                </Table.Th>
                                                <Table.Th ta="right">
                                                    <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                        %
                                                    </Text>
                                                </Table.Th>
                                            </Table.Tr>
                                        </Table.Thead>
                                        <Table.Tbody>
                                            {mediosPago.map((m, idx) => {
                                                const pct =
                                                    totalMediosPago > 0
                                                        ? Math.round((m.total / totalMediosPago) * 100)
                                                        : 0;
                                                const color = MEDIO_PAGO_COLOR[m.medio_pago] ?? 'gray';
                                                return (
                                                    <Table.Tr key={idx}>
                                                        <Table.Td>
                                                            <Badge
                                                                variant="light"
                                                                color={color}
                                                                size="sm"
                                                                tt="capitalize"
                                                            >
                                                                {m.medio_pago}
                                                            </Badge>
                                                        </Table.Td>
                                                        <Table.Td ta="right">
                                                            <Text size="sm" c="dimmed">
                                                                {m.cantidad}
                                                            </Text>
                                                        </Table.Td>
                                                        <Table.Td ta="right">
                                                            <Text size="sm" fw={700}>
                                                                {formatARS(m.total)}
                                                            </Text>
                                                        </Table.Td>
                                                        <Table.Td ta="right">
                                                            <Badge variant="filled" color={color} size="sm">
                                                                {pct}%
                                                            </Badge>
                                                        </Table.Td>
                                                    </Table.Tr>
                                                );
                                            })}
                                        </Table.Tbody>

                                        {/* Total row */}
                                        {mediosPago.length > 0 && (
                                            <Table.Tfoot>
                                                <Table.Tr
                                                    style={{
                                                        borderTop: '2px solid var(--mantine-color-default-border)',
                                                    }}
                                                >
                                                    <Table.Td>
                                                        <Text size="sm" fw={700}>
                                                            Total
                                                        </Text>
                                                    </Table.Td>
                                                    <Table.Td ta="right">
                                                        <Text size="sm" fw={700} c="dimmed">
                                                            {mediosPago.reduce((s, m) => s + m.cantidad, 0)}
                                                        </Text>
                                                    </Table.Td>
                                                    <Table.Td ta="right">
                                                        <Text size="sm" fw={800}>
                                                            {formatARS(totalMediosPago)}
                                                        </Text>
                                                    </Table.Td>
                                                    <Table.Td ta="right">
                                                        <Text size="sm" fw={700} c="dimmed">
                                                            100%
                                                        </Text>
                                                    </Table.Td>
                                                </Table.Tr>
                                            </Table.Tfoot>
                                        )}
                                    </Table>
                                )}
                            </Paper>
                        </SimpleGrid>
                    </Stack>
                </Tabs.Panel>

                {/* ── Por Cajero Tab ──────────────────────────────────────────── */}
                <Tabs.Panel value="cajeros" pt="lg">
                    <Paper p="lg" radius="md" withBorder>
                        <Group justify="space-between" mb="md">
                            <div>
                                <Title order={5} c="teal.4">
                                    Rendimiento por cajero
                                </Title>
                                <Text size="xs" c="dimmed">
                                    Métricas de ventas agrupadas por cajero en el periodo
                                </Text>
                            </div>
                            <ThemeIcon variant="light" color="teal" size="md" radius="sm">
                                <User size={16} />
                            </ThemeIcon>
                        </Group>

                        {loading ? (
                            <Stack gap="xs">
                                {Array.from({ length: 5 }, (_, i) => (
                                    <Skeleton key={i} h={36} radius="sm" />
                                ))}
                            </Stack>
                        ) : cajeros.length === 0 ? (
                            <Text size="sm" c="dimmed" ta="center" py="xl">
                                Sin datos de cajeros en el periodo
                            </Text>
                        ) : (
                            <Table verticalSpacing="sm" highlightOnHover withRowBorders={false}>
                                <Table.Thead>
                                    <Table.Tr
                                        style={{
                                            borderBottom: '1px solid var(--mantine-color-default-border)',
                                        }}
                                    >
                                        <Table.Th>
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Cajero
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Total Ventas
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Cantidad
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Ticket Promedio
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Descuentos
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Anulaciones
                                            </Text>
                                        </Table.Th>
                                    </Table.Tr>
                                </Table.Thead>
                                <Table.Tbody>
                                    {cajeros.map((c, idx) => (
                                        <Table.Tr key={c.usuario_id}>
                                            <Table.Td>
                                                <Group gap="xs" wrap="nowrap">
                                                    <Text size="sm" fw={500}>
                                                        {c.nombre_cajero}
                                                    </Text>
                                                    {idx === 0 && (
                                                        <Badge variant="light" color="teal" size="xs">
                                                            Top
                                                        </Badge>
                                                    )}
                                                </Group>
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <Text size="sm" fw={700}>
                                                    {formatARS(c.total_ventas)}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <Text size="sm" c="dimmed">
                                                    {c.cantidad_ventas}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <Text size="sm" c="dimmed">
                                                    {formatARS(c.ticket_promedio)}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <Text size="sm" c="dimmed">
                                                    {formatARS(c.total_descuentos)}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <Text
                                                    size="sm"
                                                    fw={c.cantidad_anulaciones > 0 ? 700 : 400}
                                                    c={c.cantidad_anulaciones > 0 ? 'red' : 'dimmed'}
                                                >
                                                    {c.cantidad_anulaciones}
                                                </Text>
                                            </Table.Td>
                                        </Table.Tr>
                                    ))}
                                </Table.Tbody>
                            </Table>
                        )}
                    </Paper>
                </Tabs.Panel>

                {/* ── Turnos Tab ──────────────────────────────────────────────── */}
                <Tabs.Panel value="turnos" pt="lg">
                    <Paper p="lg" radius="md" withBorder>
                        <Group justify="space-between" mb="md">
                            <div>
                                <Title order={5} c="orange.4">
                                    Sesiones de caja (turnos)
                                </Title>
                                <Text size="xs" c="dimmed">
                                    Historial de turnos con totales y control de desvío
                                </Text>
                            </div>
                            <ThemeIcon variant="light" color="orange" size="md" radius="sm">
                                <Clock size={16} />
                            </ThemeIcon>
                        </Group>

                        {loading ? (
                            <Stack gap="xs">
                                {Array.from({ length: 5 }, (_, i) => (
                                    <Skeleton key={i} h={36} radius="sm" />
                                ))}
                            </Stack>
                        ) : turnos.length === 0 ? (
                            <Text size="sm" c="dimmed" ta="center" py="xl">
                                Sin turnos en el periodo seleccionado
                            </Text>
                        ) : (
                            <Table verticalSpacing="sm" highlightOnHover withRowBorders={false}>
                                <Table.Thead>
                                    <Table.Tr
                                        style={{
                                            borderBottom: '1px solid var(--mantine-color-default-border)',
                                        }}
                                    >
                                        <Table.Th>
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Cajero
                                            </Text>
                                        </Table.Th>
                                        <Table.Th>
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Apertura
                                            </Text>
                                        </Table.Th>
                                        <Table.Th>
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Cierre
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Total Ventas
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Cantidad
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="right">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Desvío
                                            </Text>
                                        </Table.Th>
                                        <Table.Th ta="center">
                                            <Text size="xs" c="dimmed" fw={700} tt="uppercase">
                                                Estado
                                            </Text>
                                        </Table.Th>
                                    </Table.Tr>
                                </Table.Thead>
                                <Table.Tbody>
                                    {turnos.map((t) => {
                                        const desvioColor =
                                            t.desvio_clasificacion === 'critico'
                                                ? 'red'
                                                : t.desvio_clasificacion === 'advertencia'
                                                  ? 'yellow'
                                                  : 'green';
                                        const formatDate = (iso: string) => {
                                            const d = new Date(iso);
                                            return d.toLocaleString('es-AR', {
                                                day: '2-digit',
                                                month: '2-digit',
                                                hour: '2-digit',
                                                minute: '2-digit',
                                            });
                                        };
                                        return (
                                            <Table.Tr key={t.sesion_id}>
                                                <Table.Td>
                                                    <Text size="sm" fw={500}>
                                                        {t.cajero_nombre}
                                                    </Text>
                                                </Table.Td>
                                                <Table.Td>
                                                    <Text size="sm" c="dimmed">
                                                        {formatDate(t.fecha_apertura)}
                                                    </Text>
                                                </Table.Td>
                                                <Table.Td>
                                                    <Text size="sm" c="dimmed">
                                                        {t.fecha_cierre ? formatDate(t.fecha_cierre) : '—'}
                                                    </Text>
                                                </Table.Td>
                                                <Table.Td ta="right">
                                                    <Text size="sm" fw={700}>
                                                        {formatARS(t.total_ventas)}
                                                    </Text>
                                                </Table.Td>
                                                <Table.Td ta="right">
                                                    <Text size="sm" c="dimmed">
                                                        {t.cantidad_ventas}
                                                    </Text>
                                                </Table.Td>
                                                <Table.Td ta="right">
                                                    <Text size="sm" fw={700} c={desvioColor}>
                                                        {formatARS(t.desvio)}
                                                    </Text>
                                                </Table.Td>
                                                <Table.Td ta="center">
                                                    <Badge
                                                        variant="light"
                                                        color={desvioColor}
                                                        size="sm"
                                                        tt="capitalize"
                                                    >
                                                        {t.desvio_clasificacion}
                                                    </Badge>
                                                </Table.Td>
                                            </Table.Tr>
                                        );
                                    })}
                                </Table.Tbody>
                            </Table>
                        )}
                    </Paper>
                </Tabs.Panel>
            </Tabs>
        </Stack>
    );
}
