import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
    Paper, Title, Text, TextInput, PasswordInput,
    Button, Stack, Alert, Anchor, SimpleGrid,
    Card, Group, Badge, ThemeIcon,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle, Store, Beef, ShoppingCart, Apple, Check, ArrowLeft } from 'lucide-react';
import { registerTenant, listarPresets, type PresetResponse } from '../services/api/tenant';
import { tokenStore } from '../store/tokenStore';
import { useAuthStore } from '../store/useAuthStore';
import { BrandMark } from '../components/BrandMark';
import classes from './RegisterPage.module.css';

// ── Manrope font injection ────────────────────────────────────────────────────
const MANROPE_HREF = 'https://fonts.googleapis.com/css2?family=Manrope:wght@300;400;500;600;700;800&display=swap';
function ensureManrope() {
    if (typeof document === 'undefined') return;
    if (document.querySelector(`link[href="${MANROPE_HREF}"]`)) return;
    const link = document.createElement('link');
    link.rel = 'stylesheet'; link.href = MANROPE_HREF;
    document.head.appendChild(link);
}
ensureManrope();

// ── Brand icon ────────────────────────────────────────────────────────────────
function BrandIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 22 22" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="2" y="2" width="8" height="8" rx="2" fill="white" fillOpacity="0.9" />
            <rect x="12" y="2" width="8" height="8" rx="2" fill="white" fillOpacity="0.55" />
            <rect x="2" y="12" width="8" height="8" rx="2" fill="white" fillOpacity="0.55" />
            <rect x="12" y="12" width="8" height="8" rx="2" fill="white" fillOpacity="0.3" />
        </svg>
    );
}

// ── Business type config ────────────────────────────────────────────────────

const BUSINESS_TYPE_ICONS: Record<string, React.ReactNode> = {
    kiosco: <Store size={28} />,
    carniceria: <Beef size={28} />,
    minimarket: <ShoppingCart size={28} />,
    verduleria: <Apple size={28} />,
};

const BUSINESS_TYPE_COLORS: Record<string, string> = {
    kiosco: 'blue',
    carniceria: 'red',
    minimarket: 'teal',
    verduleria: 'green',
};

const BUSINESS_TYPE_DESCRIPTIONS: Record<string, string> = {
    kiosco: 'Golosinas, bebidas, cigarrillos y más',
    carniceria: 'Cortes, embutidos y fiambres',
    minimarket: 'Almacén con variedad de productos',
    verduleria: 'Frutas, verduras y productos frescos',
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
            const msg = e instanceof Error ? e.message : 'Error al registrar. Intentá nuevamente.';
            setError(msg);
        } finally {
            setLoading(false);
        }
    });

    const selectedPreset = presets.find((p) => p.tipo_negocio === selectedType);

    return (
        <div className={classes.wrapper}>
            {/* Back to login button */}
            <Link to="/login" className={classes.backButton}>
                <ArrowLeft size={13} />
                Volver al login
            </Link>

            <div className={classes.container}>
                {/* Branding */}
                <div className={`${classes.branding} ${classes.fadeIn}`}>
                    <BrandMark layout="col" size={52} />
                    <p className={classes.brandingTagline}>
                        Creá tu cuenta — es gratis para empezar
                    </p>
                </div>

                {/* Step indicator */}
                <div className={`${classes.stepIndicator} ${classes.fadeIn}`}>
                    <div className={`${classes.stepDot} ${classes.stepDotActive}`} />
                    <div className={classes.stepDot} />
                    <div className={classes.stepDot} />
                    <span className={classes.stepLabel}>Paso 1 de 3 — Registro</span>
                </div>

                {/* Main form card */}
                <Paper
                    p="xl"
                    radius="md"
                    withBorder
                    className={`${classes.mainCard} ${classes.fadeInDelay}`}
                >
                    {error && (
                        <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                            {error}
                        </Alert>
                    )}

                    {/* Section 1: Business type */}
                    <div className={classes.sectionTitle}>
                        <div className={classes.sectionNumber}>1</div>
                        <span className={classes.sectionLabel}>Elegí tu tipo de negocio</span>
                    </div>

                    <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="sm" mb="md">
                        {presets.map((preset) => {
                            const isSelected = selectedType === preset.tipo_negocio;
                            const color = BUSINESS_TYPE_COLORS[preset.tipo_negocio] ?? 'blue';
                            return (
                                <Card
                                    key={preset.tipo_negocio}
                                    radius="md"
                                    withBorder
                                    className={`${classes.typeCard} ${isSelected ? classes.typeCardSelected : ''}`}
                                    style={{
                                        borderColor: isSelected ? `var(--mantine-color-${color}-5)` : undefined,
                                        background: isSelected ? `var(--mantine-color-${color}-light)` : undefined,
                                    }}
                                    onClick={() => setSelectedType(preset.tipo_negocio)}
                                >
                                    <Group gap="sm" wrap="nowrap" align="flex-start">
                                        <ThemeIcon
                                            size={44}
                                            radius="md"
                                            color={color}
                                            variant={isSelected ? 'filled' : 'light'}
                                        >
                                            {BUSINESS_TYPE_ICONS[preset.tipo_negocio] ?? <Store size={28} />}
                                        </ThemeIcon>
                                        <div style={{ flex: 1, minWidth: 0 }}>
                                            <Group gap={6} align="center">
                                                <Text fw={600} size="sm">{preset.label}</Text>
                                                {isSelected && (
                                                    <ThemeIcon size={18} radius="xl" color={color} variant="filled">
                                                        <Check size={11} />
                                                    </ThemeIcon>
                                                )}
                                            </Group>
                                            <Text size="xs" c="dimmed" mt={2}>
                                                {BUSINESS_TYPE_DESCRIPTIONS[preset.tipo_negocio] ??
                                                    `${preset.total_categorias} categorías`}
                                            </Text>
                                            <Text size="xs" c="dimmed" mt={2}>
                                                {preset.total_categorias} categorías, {preset.total_productos} productos
                                            </Text>
                                        </div>
                                    </Group>
                                </Card>
                            );
                        })}
                    </SimpleGrid>

                    {selectedPreset && selectedPreset.categorias.length > 0 && (
                        <div className={classes.categoriesPreview}>
                            <Text size="xs" fw={500} mb={6} c="dimmed">Categorías incluidas:</Text>
                            <Group gap={4} wrap="wrap">
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
                        </div>
                    )}

                    <div className={classes.divider} />

                    {/* Section 2: Business info */}
                    <div className={classes.sectionTitle}>
                        <div className={classes.sectionNumber}>2</div>
                        <span className={classes.sectionLabel}>Datos del negocio</span>
                    </div>

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
                                description="Solo letras minúsculas y números, sin espacios. Será tu URL."
                                required
                                {...form.getInputProps('slug')}
                                onChange={(e) => form.setFieldValue('slug', e.currentTarget.value.toLowerCase().replace(/[^a-z0-9]/g, ''))}
                            />

                            <div className={classes.divider} />

                            {/* Section 3: User account */}
                            <div className={classes.sectionTitle}>
                                <div className={classes.sectionNumber}>3</div>
                                <span className={classes.sectionLabel}>Tu cuenta de administrador</span>
                            </div>

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
                            <Button type="submit" fullWidth loading={loading} mt="sm" size="md">
                                Crear cuenta gratis
                            </Button>
                        </Stack>
                    </form>

                    <Text size="sm" c="dimmed" ta="center" mt="lg">
                        ¿Ya tenés cuenta?{' '}
                        <Anchor component={Link} to="/login" size="sm" fw={600}>
                            Iniciá sesión
                        </Anchor>
                    </Text>
                </Paper>
            </div>
        </div>
    );
}
