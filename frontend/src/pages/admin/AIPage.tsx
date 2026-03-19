// ─────────────────────────────────────────────────────────────────────────────
// AIPage — Asistente IA con chat y métricas pre-calculadas
// Powered by Mistral AI via backend proxy
// ─────────────────────────────────────────────────────────────────────────────

import { useCallback, useEffect, useRef, useState } from 'react';
import {
    ActionIcon,
    Alert,
    Badge,
    Button,
    Group,
    Loader,
    Paper,
    ScrollArea,
    SimpleGrid,
    Skeleton,
    Stack,
    Tabs,
    Text,
    TextInput,
    ThemeIcon,
    Title,
    Tooltip,
} from '@mantine/core';
import {
    AlertTriangle,
    ArrowUp,
    Bot,
    BrainCircuit,
    Clock,
    Package,
    Send,
    ShoppingCart,
    Sparkles,
    TrendingDown,
    TrendingUp,
    User,
    Zap,
} from 'lucide-react';
import { formatARS } from '../../utils/format';
import {
    chatAI,
    getMetricasAI,
    getAIStatus,
    type AIChatResponse,
    type AIMetricasResponse,
    type AIStatusResponse,
} from '../../services/api/ai';

// ── Types ───────────────────────────────────────────────────────────────────

interface ChatMessage {
    role: 'user' | 'assistant';
    content: string;
    timestamp: Date;
}

// ── KPI Card (reused pattern from ReportesPage) ─────────────────────────────

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

// ── Chat Bubble ─────────────────────────────────────────────────────────────

function ChatBubble({ message }: { message: ChatMessage }) {
    const isUser = message.role === 'user';

    return (
        <Group
            justify={isUser ? 'flex-end' : 'flex-start'}
            align="flex-start"
            gap="sm"
            wrap="nowrap"
        >
            {!isUser && (
                <ThemeIcon variant="light" color="violet" size="md" radius="xl">
                    <Bot size={14} />
                </ThemeIcon>
            )}
            <Paper
                p="sm"
                radius="md"
                withBorder
                maw="75%"
                style={{
                    backgroundColor: isUser
                        ? 'var(--mantine-color-blue-light)'
                        : 'var(--mantine-color-default)',
                    borderColor: isUser
                        ? 'var(--mantine-color-blue-3)'
                        : 'var(--mantine-color-default-border)',
                }}
            >
                <Text
                    size="sm"
                    style={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}
                >
                    {message.content}
                </Text>
                <Text size="xs" c="dimmed" ta="right" mt={4}>
                    {message.timestamp.toLocaleTimeString('es-AR', {
                        hour: '2-digit',
                        minute: '2-digit',
                    })}
                </Text>
            </Paper>
            {isUser && (
                <ThemeIcon variant="light" color="blue" size="md" radius="xl">
                    <User size={14} />
                </ThemeIcon>
            )}
        </Group>
    );
}

// ── Main Page ───────────────────────────────────────────────────────────────

export function AIPage() {
    const [activeTab, setActiveTab] = useState<string | null>('chat');

    // Status
    const [status, setStatus] = useState<AIStatusResponse | null>(null);
    const [statusLoading, setStatusLoading] = useState(true);

    // Chat state
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [inputValue, setInputValue] = useState('');
    const [chatLoading, setChatLoading] = useState(false);
    const [chatError, setChatError] = useState<string | null>(null);
    const scrollRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    // Metricas state
    const [metricas, setMetricas] = useState<AIMetricasResponse | null>(null);
    const [metricasLoading, setMetricasLoading] = useState(false);
    const [metricasError, setMetricasError] = useState<string | null>(null);

    // Check AI status on mount
    useEffect(() => {
        getAIStatus()
            .then(setStatus)
            .catch(() => setStatus({ configured: false, model: '' }))
            .finally(() => setStatusLoading(false));
    }, []);

    // Auto-scroll chat to bottom
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
        }
    }, [messages]);

    // ── Chat handlers ───────────────────────────────────────────────────────

    const handleSend = useCallback(async () => {
        const trimmed = inputValue.trim();
        if (!trimmed || chatLoading) return;

        const userMsg: ChatMessage = {
            role: 'user',
            content: trimmed,
            timestamp: new Date(),
        };

        setMessages((prev) => [...prev, userMsg]);
        setInputValue('');
        setChatLoading(true);
        setChatError(null);

        try {
            const resp: AIChatResponse = await chatAI(trimmed);
            const aiMsg: ChatMessage = {
                role: 'assistant',
                content: resp.response,
                timestamp: new Date(),
            };
            setMessages((prev) => [...prev, aiMsg]);
        } catch (err) {
            setChatError(String(err));
        } finally {
            setChatLoading(false);
            inputRef.current?.focus();
        }
    }, [inputValue, chatLoading]);

    const handleKeyDown = useCallback(
        (e: React.KeyboardEvent) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSend();
            }
        },
        [handleSend],
    );

    // ── Metricas handler ────────────────────────────────────────────────────

    const handleGenerateMetricas = useCallback(async () => {
        setMetricasLoading(true);
        setMetricasError(null);
        try {
            const resp = await getMetricasAI();
            setMetricas(resp);
        } catch (err) {
            setMetricasError(String(err));
        } finally {
            setMetricasLoading(false);
        }
    }, []);

    // ── Render ──────────────────────────────────────────────────────────────

    const notConfigured = !statusLoading && status && !status.configured;

    return (
        <Stack gap="xl">
            {/* Header */}
            <Group justify="space-between" align="flex-end" wrap="wrap">
                <div>
                    <Group gap="sm" align="center">
                        <ThemeIcon variant="gradient" gradient={{ from: 'violet', to: 'grape', deg: 135 }} size="lg" radius="md">
                            <BrainCircuit size={20} />
                        </ThemeIcon>
                        <Title order={2} fw={800} c="violet.4">
                            Asistente IA
                        </Title>
                    </Group>
                    <Text c="dimmed" size="sm" mt={4}>
                        Preguntale lo que quieras sobre tu negocio
                    </Text>
                </div>
                <Group gap="sm">
                    {statusLoading ? (
                        <Skeleton w={120} h={24} radius="xl" />
                    ) : (
                        <Tooltip label={status?.configured ? `Modelo: ${status.model}` : 'Sin API Key configurada'}>
                            <Badge
                                variant="light"
                                color={status?.configured ? 'green' : 'red'}
                                size="lg"
                                leftSection={
                                    status?.configured ? <Zap size={12} /> : <AlertTriangle size={12} />
                                }
                            >
                                {status?.configured ? 'Conectado' : 'Sin API Key'}
                            </Badge>
                        </Tooltip>
                    )}
                    <Text size="xs" c="dimmed">
                        Powered by Mistral AI
                    </Text>
                </Group>
            </Group>

            {/* Not configured alert */}
            {notConfigured && (
                <Alert
                    icon={<AlertTriangle size={18} />}
                    color="orange"
                    title="IA no configurada"
                >
                    Para usar el asistente IA, configura la variable de entorno{' '}
                    <Text component="span" fw={700} ff="monospace" size="sm">
                        MISTRAL_API_KEY
                    </Text>{' '}
                    en el servidor. Podés obtener una key en{' '}
                    <Text component="a" href="https://console.mistral.ai/" target="_blank" c="blue" td="underline" size="sm">
                        console.mistral.ai
                    </Text>
                </Alert>
            )}

            {/* Tabs */}
            <Tabs value={activeTab} onChange={setActiveTab}>
                <Tabs.List>
                    <Tabs.Tab value="chat" leftSection={<Bot size={16} />}>
                        Chat
                    </Tabs.Tab>
                    <Tabs.Tab value="metricas" leftSection={<Sparkles size={16} />}>
                        Metricas
                    </Tabs.Tab>
                </Tabs.List>

                {/* ── Chat Tab ──────────────────────────────────────────────── */}
                <Tabs.Panel value="chat" pt="lg">
                    <Paper
                        radius="md"
                        withBorder
                        style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 320px)', minHeight: 400 }}
                    >
                        {/* Messages */}
                        <ScrollArea
                            flex={1}
                            p="md"
                            viewportRef={scrollRef}
                        >
                            {messages.length === 0 ? (
                                <Stack align="center" justify="center" h="100%" gap="md" py="xl">
                                    <ThemeIcon variant="light" color="violet" size={60} radius="xl">
                                        <BrainCircuit size={30} />
                                    </ThemeIcon>
                                    <Text size="lg" fw={600} c="dimmed" ta="center">
                                        Preguntale algo a tu asistente
                                    </Text>
                                    <Stack gap="xs" align="center">
                                        {[
                                            'Como fueron las ventas este mes?',
                                            'Cuales son mis productos estrella?',
                                            'Que horarios son los mas fuertes?',
                                            'Tengo productos con stock bajo?',
                                        ].map((suggestion) => (
                                            <Button
                                                key={suggestion}
                                                variant="light"
                                                color="violet"
                                                size="xs"
                                                radius="xl"
                                                onClick={() => {
                                                    setInputValue(suggestion);
                                                    inputRef.current?.focus();
                                                }}
                                            >
                                                {suggestion}
                                            </Button>
                                        ))}
                                    </Stack>
                                </Stack>
                            ) : (
                                <Stack gap="md">
                                    {messages.map((msg, idx) => (
                                        <ChatBubble key={idx} message={msg} />
                                    ))}
                                    {chatLoading && (
                                        <Group gap="sm" align="center">
                                            <ThemeIcon variant="light" color="violet" size="md" radius="xl">
                                                <Bot size={14} />
                                            </ThemeIcon>
                                            <Paper p="sm" radius="md" withBorder>
                                                <Group gap="xs">
                                                    <Loader size="xs" color="violet" />
                                                    <Text size="sm" c="dimmed">Pensando...</Text>
                                                </Group>
                                            </Paper>
                                        </Group>
                                    )}
                                </Stack>
                            )}
                        </ScrollArea>

                        {/* Error */}
                        {chatError && (
                            <Alert
                                icon={<AlertTriangle size={16} />}
                                color="red"
                                m="sm"
                                withCloseButton
                                onClose={() => setChatError(null)}
                            >
                                {chatError}
                            </Alert>
                        )}

                        {/* Input */}
                        <Group
                            p="md"
                            gap="sm"
                            style={{ borderTop: '1px solid var(--mantine-color-default-border)' }}
                        >
                            <TextInput
                                ref={inputRef}
                                flex={1}
                                placeholder={notConfigured ? 'IA no configurada...' : 'Escribi tu pregunta...'}
                                value={inputValue}
                                onChange={(e) => setInputValue(e.currentTarget.value)}
                                onKeyDown={handleKeyDown}
                                disabled={!!notConfigured || chatLoading}
                                size="md"
                                radius="xl"
                                rightSection={
                                    <ActionIcon
                                        variant="filled"
                                        color="violet"
                                        size="md"
                                        radius="xl"
                                        onClick={handleSend}
                                        disabled={!inputValue.trim() || chatLoading || !!notConfigured}
                                        aria-label="Enviar mensaje"
                                    >
                                        <Send size={14} />
                                    </ActionIcon>
                                }
                            />
                        </Group>
                    </Paper>
                </Tabs.Panel>

                {/* ── Metricas Tab ──────────────────────────────────────────── */}
                <Tabs.Panel value="metricas" pt="lg">
                    <Stack gap="xl">
                        {/* Generate button */}
                        {!metricas && !metricasLoading && (
                            <Paper p="xl" radius="md" withBorder ta="center">
                                <Stack align="center" gap="md">
                                    <ThemeIcon variant="light" color="violet" size={60} radius="xl">
                                        <Sparkles size={30} />
                                    </ThemeIcon>
                                    <Text size="lg" fw={600}>
                                        Analisis inteligente de tu negocio
                                    </Text>
                                    <Text size="sm" c="dimmed" maw={400}>
                                        Genera un analisis completo con metricas clave e insights
                                        accionables basados en los datos de tu negocio.
                                    </Text>
                                    <Button
                                        variant="gradient"
                                        gradient={{ from: 'violet', to: 'grape', deg: 135 }}
                                        size="md"
                                        leftSection={<Sparkles size={18} />}
                                        onClick={handleGenerateMetricas}
                                        disabled={!!notConfigured}
                                    >
                                        Generar Analisis
                                    </Button>
                                </Stack>
                            </Paper>
                        )}

                        {/* Error */}
                        {metricasError && (
                            <Alert
                                icon={<AlertTriangle size={18} />}
                                color="red"
                                title="Error al generar metricas"
                                withCloseButton
                                onClose={() => setMetricasError(null)}
                            >
                                {metricasError}
                            </Alert>
                        )}

                        {/* Loading */}
                        {metricasLoading && (
                            <Stack gap="md">
                                <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
                                    {Array.from({ length: 4 }, (_, i) => (
                                        <Skeleton key={i} h={110} radius="md" />
                                    ))}
                                </SimpleGrid>
                                <Skeleton h={200} radius="md" />
                            </Stack>
                        )}

                        {/* Results */}
                        {metricas && !metricasLoading && (
                            <>
                                {/* KPI Cards */}
                                <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
                                    <KpiCard
                                        label="Ventas del mes"
                                        value={formatARS(metricas.ventas_mes_actual)}
                                        sub={`${metricas.cantidad_ventas} transacciones`}
                                        icon={<ShoppingCart size={22} />}
                                        color="teal"
                                        loading={false}
                                    />
                                    <KpiCard
                                        label="Ticket promedio"
                                        value={formatARS(metricas.ticket_promedio)}
                                        sub="Promedio por venta"
                                        icon={<TrendingUp size={22} />}
                                        color="blue"
                                        loading={false}
                                    />
                                    <KpiCard
                                        label="vs. mes anterior"
                                        value={`${metricas.variacion_porcentaje >= 0 ? '+' : ''}${metricas.variacion_porcentaje.toFixed(1)}%`}
                                        sub={formatARS(metricas.ventas_mes_anterior)}
                                        icon={metricas.variacion_porcentaje >= 0 ? <ArrowUp size={22} /> : <TrendingDown size={22} />}
                                        color={metricas.variacion_porcentaje >= 0 ? 'green' : 'red'}
                                        loading={false}
                                    />
                                    <KpiCard
                                        label="Alertas stock"
                                        value={String(metricas.alertas_stock)}
                                        sub="Productos con stock bajo"
                                        icon={<Package size={22} />}
                                        color={metricas.alertas_stock > 0 ? 'orange' : 'gray'}
                                        loading={false}
                                    />
                                </SimpleGrid>

                                {/* Data panels */}
                                <SimpleGrid cols={{ base: 1, md: 2 }}>
                                    {/* Top productos */}
                                    <Paper p="lg" radius="md" withBorder>
                                        <Group justify="space-between" mb="md">
                                            <div>
                                                <Title order={5} c="teal.4">Top 5 productos</Title>
                                                <Text size="xs" c="dimmed">Mas vendidos este mes</Text>
                                            </div>
                                            <ThemeIcon variant="light" color="teal" size="md" radius="sm">
                                                <TrendingUp size={16} />
                                            </ThemeIcon>
                                        </Group>
                                        <Stack gap="xs">
                                            {(metricas.top_productos ?? []).map((p, i) => (
                                                <Group key={i} justify="space-between">
                                                    <Group gap="xs">
                                                        <Badge variant="light" color="teal" size="sm" circle>
                                                            {i + 1}
                                                        </Badge>
                                                        <Text size="sm" fw={500} lineClamp={1}>
                                                            {p.nombre}
                                                        </Text>
                                                    </Group>
                                                    <Text size="sm" fw={700}>
                                                        {formatARS(p.total)}
                                                    </Text>
                                                </Group>
                                            ))}
                                            {(!metricas.top_productos || metricas.top_productos.length === 0) && (
                                                <Text size="sm" c="dimmed" ta="center" py="md">Sin datos</Text>
                                            )}
                                        </Stack>
                                    </Paper>

                                    {/* Horas pico */}
                                    <Paper p="lg" radius="md" withBorder>
                                        <Group justify="space-between" mb="md">
                                            <div>
                                                <Title order={5} c="orange.4">Horas pico</Title>
                                                <Text size="xs" c="dimmed">Horarios con mas ventas</Text>
                                            </div>
                                            <ThemeIcon variant="light" color="orange" size="md" radius="sm">
                                                <Clock size={16} />
                                            </ThemeIcon>
                                        </Group>
                                        <Stack gap="xs">
                                            {(metricas.horas_pico ?? []).map((h, i) => (
                                                <Group key={i} justify="space-between">
                                                    <Text size="sm" fw={500}>
                                                        {String(h.hora).padStart(2, '0')}:00 hs
                                                    </Text>
                                                    <Badge variant="light" color="orange" size="sm">
                                                        {h.cantidad} ventas
                                                    </Badge>
                                                </Group>
                                            ))}
                                            {(!metricas.horas_pico || metricas.horas_pico.length === 0) && (
                                                <Text size="sm" c="dimmed" ta="center" py="md">Sin datos</Text>
                                            )}
                                        </Stack>
                                    </Paper>
                                </SimpleGrid>

                                {/* AI Analysis */}
                                <Paper
                                    p="lg"
                                    radius="md"
                                    withBorder
                                    style={{
                                        borderColor: 'var(--mantine-color-violet-3)',
                                        background: 'var(--mantine-color-violet-light)',
                                    }}
                                >
                                    <Group gap="sm" mb="md">
                                        <ThemeIcon variant="gradient" gradient={{ from: 'violet', to: 'grape', deg: 135 }} size="md" radius="sm">
                                            <Sparkles size={16} />
                                        </ThemeIcon>
                                        <Title order={5} c="violet.4">
                                            Analisis IA
                                        </Title>
                                    </Group>
                                    <Text
                                        size="sm"
                                        style={{ whiteSpace: 'pre-wrap', lineHeight: 1.8 }}
                                    >
                                        {metricas.analisis_ia}
                                    </Text>
                                </Paper>

                                {/* Regenerate button */}
                                <Group justify="center">
                                    <Button
                                        variant="light"
                                        color="violet"
                                        leftSection={<Sparkles size={16} />}
                                        onClick={handleGenerateMetricas}
                                    >
                                        Regenerar analisis
                                    </Button>
                                </Group>
                            </>
                        )}
                    </Stack>
                </Tabs.Panel>
            </Tabs>
        </Stack>
    );
}
