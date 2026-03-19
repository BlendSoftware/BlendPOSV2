import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
    Center, Paper, Title, Text, TextInput, PasswordInput,
    Button, Stack, Alert, Box, Anchor, SimpleGrid,
    Card, Group, Badge, ThemeIcon,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle, Store, Beef, ShoppingCart, Apple, Check } from 'lucide-react';
import { registerTenant, listarPresets, type PresetResponse } from '../services/api/tenant';
import { tokenStore } from '../store/tokenStore';
import { useAuthStore } from '../store/useAuthStore';

// ── Business type config ────────────────────────────────────────────────────

const BUSINESS_TYPE_ICONS: Record<string, React.ReactNode> = {
    kiosco: <Store size={24} />,
    carniceria: <Beef size={24} />,
    minimarket: <ShoppingCart size={24} />,
    verduleria: <Apple size={24} />,
};

const BUSINESS_TYPE_COLORS: Record<string, string> = {
    kiosco: 'blue',
    carniceria: 'red',
    minimarket: 'teal',
    verduleria: 'green',
};

export function RegisterPage() {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [presets, setPresets] = useState<PresetResponse[]>([]);
    const [selectedType, setSelectedType] = useState('kiosco');

    useEffect(() => {
        listarPresets()
            .then(setPresets)
            .catch(() => {
                // Presets are non-critical — fallback to defaults
                setPresets([
                    { tipo_negocio: 'kiosco', label: 'Kiosco', total_categorias: 8, total_productos: 8, categorias: [] },
                    { tipo_negocio: 'carniceria', label: 'Carnicería', total_categorias: 6, total_productos: 8, categorias: [] },
                    { tipo_negocio: 'minimarket', label: 'Minimarket', total_categorias: 8, total_productos: 8, categorias: [] },
                    { tipo_negocio: 'verduleria', label: 'Verdulería', total_categorias: 4, total_productos: 7, categorias: [] },
                ]);
            });
    }, []);

    const form = useForm({
        initialValues: {
            nombre_negocio: '',
            slug: '',
            nombre: '',
            username: '',
            email: '',
            password: '',
            confirm_password: '',
        },
        validate: {
            nombre_negocio: (v) => (v.trim().length >= 2 ? null : 'Mínimo 2 caracteres'),
            slug: (v) => (/^[a-z0-9]{2,63}$/.test(v) ? null : 'Solo letras minúsculas y números, 2-63 caracteres'),
            nombre: (v) => (v.trim().length >= 2 ? null : 'Mínimo 2 caracteres'),
            username: (v) => (v.trim().length >= 3 ? null : 'Mínimo 3 caracteres'),
            password: (v) => (v.length >= 8 ? null : 'Mínimo 8 caracteres'),
            confirm_password: (v, values) => (v === values.password ? null : 'Las contraseñas no coinciden'),
            email: (v) => (!v || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) ? null : 'Email inválido'),
        },
    });

    const handleSubmit = form.onSubmit(async (values) => {
        setError('');
        setLoading(true);
        try {
            const resp = await registerTenant({
                nombre_negocio: values.nombre_negocio,
                slug: values.slug.toLowerCase(),
                nombre: values.nombre,
                username: values.username,
                password: values.password,
                email: values.email || undefined,
                tipo_negocio: selectedType,
            });
            // Log in immediately — tokens issued by registration
            tokenStore.setTokens(resp.access_token, resp.refresh_token);
            // Sync auth store state via initAuth (reads token from memory)
            await useAuthStore.getState().refresh();
            navigate('/onboarding', { replace: true });
        } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : 'Error al registrar. Intente nuevamente.';
            setError(msg);
        } finally {
            setLoading(false);
        }
    });

    const selectedPreset = presets.find((p) => p.tipo_negocio === selectedType);

    return (
        <Center style={{ minHeight: '100vh', background: 'var(--mantine-color-body)', padding: '2rem 0' }}>
            <Box w={520}>
                <Stack gap="xs" mb="xl" align="center">
                    <Title order={1} c="blue.4" fw={800} style={{ letterSpacing: '-1px' }}>
                        BlendPOS
                    </Title>
                    <Text c="dimmed" size="sm">Creá tu cuenta — es gratis para empezar</Text>
                </Stack>

                <Paper p="xl" radius="md" withBorder>
                    <Title order={3} mb="lg">Registrar negocio</Title>

                    {error && (
                        <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                            {error}
                        </Alert>
                    )}

                    {/* Business type selection */}
                    <Text fw={500} size="sm" mb="xs">Tipo de negocio</Text>
                    <SimpleGrid cols={2} spacing="sm" mb="md">
                        {presets.map((preset) => {
                            const isSelected = selectedType === preset.tipo_negocio;
                            const color = BUSINESS_TYPE_COLORS[preset.tipo_negocio] ?? 'blue';
                            return (
                                <Card
                                    key={preset.tipo_negocio}
                                    padding="sm"
                                    radius="md"
                                    withBorder
                                    style={{
                                        cursor: 'pointer',
                                        borderColor: isSelected ? `var(--mantine-color-${color}-5)` : undefined,
                                        borderWidth: isSelected ? 2 : 1,
                                        background: isSelected ? `var(--mantine-color-${color}-0)` : undefined,
                                    }}
                                    onClick={() => setSelectedType(preset.tipo_negocio)}
                                >
                                    <Group gap="xs" wrap="nowrap">
                                        <ThemeIcon
                                            size="lg"
                                            radius="md"
                                            color={color}
                                            variant={isSelected ? 'filled' : 'light'}
                                        >
                                            {BUSINESS_TYPE_ICONS[preset.tipo_negocio] ?? <Store size={24} />}
                                        </ThemeIcon>
                                        <div style={{ flex: 1, minWidth: 0 }}>
                                            <Group gap={4} align="center">
                                                <Text fw={600} size="sm">{preset.label}</Text>
                                                {isSelected && (
                                                    <ThemeIcon size={16} radius="xl" color={color} variant="filled">
                                                        <Check size={10} />
                                                    </ThemeIcon>
                                                )}
                                            </Group>
                                            <Text size="xs" c="dimmed">
                                                {preset.total_categorias} categorías, {preset.total_productos} productos
                                            </Text>
                                        </div>
                                    </Group>
                                </Card>
                            );
                        })}
                    </SimpleGrid>

                    {selectedPreset && selectedPreset.categorias.length > 0 && (
                        <Group gap={4} mb="md" wrap="wrap">
                            {selectedPreset.categorias.map((cat) => (
                                <Badge
                                    key={cat.nombre}
                                    size="xs"
                                    variant="light"
                                    color={BUSINESS_TYPE_COLORS[selectedType] ?? 'blue'}
                                >
                                    {cat.nombre}
                                </Badge>
                            ))}
                        </Group>
                    )}

                    <form onSubmit={handleSubmit}>
                        <Stack gap="md">
                            <TextInput
                                label="Nombre del negocio"
                                placeholder="Mi Kiosco"
                                required
                                {...form.getInputProps('nombre_negocio')}
                                data-autofocus
                            />
                            <TextInput
                                label="Identificador único (slug)"
                                placeholder="mikiosco"
                                description="Solo letras minúsculas y números, sin espacios"
                                required
                                {...form.getInputProps('slug')}
                                onChange={(e) => form.setFieldValue('slug', e.currentTarget.value.toLowerCase().replace(/[^a-z0-9]/g, ''))}
                            />
                            <TextInput
                                label="Tu nombre"
                                placeholder="Juan Pérez"
                                required
                                {...form.getInputProps('nombre')}
                            />
                            <TextInput
                                label="Usuario administrador"
                                placeholder="admin"
                                required
                                {...form.getInputProps('username')}
                            />
                            <TextInput
                                label="Email (opcional)"
                                placeholder="juan@mikiosco.com"
                                type="email"
                                {...form.getInputProps('email')}
                            />
                            <PasswordInput
                                label="Contraseña"
                                placeholder="Mínimo 8 caracteres"
                                required
                                {...form.getInputProps('password')}
                            />
                            <PasswordInput
                                label="Confirmar contraseña"
                                placeholder="Repetir contraseña"
                                required
                                {...form.getInputProps('confirm_password')}
                            />
                            <Button type="submit" fullWidth loading={loading} mt="sm">
                                Crear cuenta
                            </Button>
                        </Stack>
                    </form>

                    <Text size="sm" c="dimmed" ta="center" mt="md">
                        ¿Ya tenés cuenta?{' '}
                        <Anchor component={Link} to="/login" size="sm">
                            Iniciá sesión
                        </Anchor>
                    </Text>
                </Paper>
            </Box>
        </Center>
    );
}
