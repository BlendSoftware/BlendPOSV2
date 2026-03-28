import { useState, useEffect } from 'react';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import {
    Paper, Title, Text, TextInput, PasswordInput,
    Button, Stack, Alert, Anchor, Checkbox, Group,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { AlertCircle, ShieldAlert, WifiOff, Building2, FileCheck, Chrome } from 'lucide-react';
import { useAuthStore } from '../../store/useAuthStore';
import { changePasswordApi } from '../../services/api/auth';
import { SupportModal } from '../../components/SupportModal';
import { TermsModal } from '../../components/TermsModal';
import { BrandMark } from '../../components/BrandMark';
import classes from './LoginPage.module.css';

// ── Manrope font — injected once ────────────────────────────────────────────
const MANROPE_HREF = 'https://fonts.googleapis.com/css2?family=Manrope:wght@300;400;500;600;700;800&display=swap';

function ensureManrope() {
    if (typeof document === 'undefined') return;
    if (document.querySelector(`link[href="${MANROPE_HREF}"]`)) return;
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = MANROPE_HREF;
    document.head.appendChild(link);
}
ensureManrope();

// ── Feature list data ────────────────────────────────────────────────────────

const FEATURES = [
    {
        icon: <WifiOff size={18} />,
        label: 'Funciona offline',
        description: 'Seguí vendiendo incluso sin conexión. Hasta 48hs de autonomía.',
    },
    {
        icon: <Building2 size={18} />,
        label: 'Multi-sucursal',
        description: 'Controlá todas tus sucursales desde un solo lugar, en tiempo real.',
    },
    {
        icon: <FileCheck size={18} />,
        label: 'AFIP integrado',
        description: 'Facturación electrónica automática, sin complicaciones.',
    },
];

// ── Component ────────────────────────────────────────────────────────────────

export function LoginPage() {
    const navigate = useNavigate();
    const location = useLocation();
    const { login, isAuthenticated, user, mustChangePassword, clearMustChangePassword } = useAuthStore();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [rememberMe, setRememberMe] = useState(true);
    const [supportOpen, setSupportOpen] = useState(false);
    const [termsOpen, setTermsOpen] = useState(false);

    const from = (location.state as { from?: { pathname: string } })?.from?.pathname ?? '/';

    // Route guard: redirect if already authenticated AND password change is not required
    useEffect(() => {
        if (isAuthenticated && !mustChangePassword) {
            if (from !== '/' && from !== '/login') {
                navigate(from, { replace: true });
            } else {
                const isAdminRole = user?.rol === 'admin' || user?.rol === 'supervisor';
                navigate(isAdminRole ? '/admin/dashboard' : '/', { replace: true });
            }
        }
    }, [isAuthenticated, user, mustChangePassword, navigate, from]);

    const form = useForm<{ email: string; password: string }>({
        initialValues: { email: '', password: '' },
        validate: {
            email: (v) => (v.trim().length >= 2 ? null : 'Ingresá usuario o email'),
            password: (v) => (v.length >= 4 ? null : 'Mínimo 4 caracteres'),
        },
    });

    const pwForm = useForm<{ newPassword: string; confirmPassword: string }>({
        initialValues: { newPassword: '', confirmPassword: '' },
        validate: {
            newPassword: (v) => (v.length >= 8 ? null : 'Mínimo 8 caracteres'),
            confirmPassword: (v, values) =>
                v === values.newPassword ? null : 'Las contraseñas no coinciden',
        },
    });

    const handleSubmit = form.onSubmit(async ({ email, password }) => {
        if (loading) return;
        setError('');
        setLoading(true);
        try {
            const ok = await login(email, password);
            if (ok) {
                const needsChange = useAuthStore.getState().mustChangePassword;
                if (!needsChange) {
                    const updatedUser = useAuthStore.getState().user;
                    const isAdminRole = updatedUser?.rol === 'admin' || updatedUser?.rol === 'supervisor';
                    const isAdminRoute = from.startsWith('/admin');
                    if (isAdminRole && !isAdminRoute) {
                        navigate('/admin/dashboard', { replace: true });
                    } else {
                        navigate(from === '/login' ? '/' : from, { replace: true });
                    }
                }
            } else {
                setError('Credenciales inválidas o usuario inactivo.');
            }
        } catch (err) {
            if (err instanceof Error && (err.name === 'OfflineError' || /fetch|network/i.test(err.message))) {
                setError('Error de conexión con el servidor. Verificá que el servidor esté corriendo.');
            } else if (err instanceof Error && /^5\d{2}\s/.test(err.message)) {
                setError('Error de conexión con el servidor. Verificá que el servidor esté corriendo.');
            } else {
                setError('Error inesperado. Intentá de nuevo.');
            }
        } finally {
            setLoading(false);
        }
    });

    const handleChangePassword = pwForm.onSubmit(async ({ newPassword }) => {
        setError('');
        setLoading(true);
        try {
            await changePasswordApi(newPassword);
            clearMustChangePassword();
            const updatedUser = useAuthStore.getState().user;
            const isAdminRole = updatedUser?.rol === 'admin' || updatedUser?.rol === 'supervisor';
            navigate(isAdminRole ? '/admin/dashboard' : '/', { replace: true });
        } catch {
            setError('Error al cambiar la contraseña. Intentá nuevamente.');
        } finally {
            setLoading(false);
        }
    });

    // ── Branding left panel ──────────────────────────────────────────────────
    const HeroPanel = (
        <div className={classes.hero}>
            <div className={`${classes.heroContent} ${classes.fadeIn}`}>
                {/* Logo */}
                <div className={classes.logoMark}>
                    <BrandMark size={52} />
                </div>

                {/* Headlines */}
                <h1 className={classes.headline}>
                    Tu POS inteligente para el comercio argentino
                </h1>
                <p className={classes.subheadline}>
                    Gestioná ventas, stock y facturación en un solo sistema
                    simple, rápido y confiable.
                </p>

                {/* Features */}
                <div className={classes.featureList}>
                    {FEATURES.map((f) => (
                        <div key={f.label} className={classes.featureItem}>
                            <div className={classes.featureIconWrap}>
                                {f.icon}
                            </div>
                            <div>
                                <div className={classes.featureLabel}>{f.label}</div>
                                <div className={classes.featureDesc}>{f.description}</div>
                            </div>
                        </div>
                    ))}
                </div>

                {/* Trust badge */}
                <div className={classes.trustBadge}>
                    <span className={classes.trustBadgeDot} />
                    Diseñado para negocios reales. Optimizado para velocidad y estabilidad.
                </div>
            </div>
        </div>
    );

    // ── SEC-03: Forced password change form ──────────────────────────────────
    if (isAuthenticated && mustChangePassword) {
        return (
            <div className={classes.wrapper}>
                {HeroPanel}

                <div className={classes.formSide}>
                    <div className={`${classes.formContainer} ${classes.fadeIn}`}>
                        {/* Mobile branding */}
                        <div className={classes.mobileBranding}>
                            <div className={classes.mobileLogo}>
                                Blend<span className={classes.mobileLogoAccent}>POS</span>
                            </div>
                            <span className={classes.mobileTagline}>Cambio de contraseña obligatorio</span>
                        </div>

                        <Paper p="xl" className={`${classes.formCard} ${classes.forceChangeCard}`}>
                            <Title order={3} className={classes.formTitle} mb="xs">
                                Cambiar contraseña
                            </Title>
                            <Text className={classes.formSubtitle}>
                                Por seguridad, debés establecer una nueva contraseña antes de continuar.
                            </Text>

                            <Alert icon={<ShieldAlert size={16} />} color="orange" mb="md" variant="light">
                                Tu contraseña temporal debe ser cambiada antes de operar.
                            </Alert>

                            {error && (
                                <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                                    {error}
                                </Alert>
                            )}

                            <form onSubmit={handleChangePassword}>
                                <Stack gap="md">
                                    <PasswordInput
                                        label="Nueva contraseña"
                                        placeholder="Mínimo 8 caracteres"
                                        {...pwForm.getInputProps('newPassword')}
                                        data-autofocus
                                    />
                                    <PasswordInput
                                        label="Confirmar contraseña"
                                        placeholder="Repetir contraseña"
                                        {...pwForm.getInputProps('confirmPassword')}
                                    />
                                    <Button
                                        type="submit"
                                        fullWidth
                                        loading={loading}
                                        mt="sm"
                                        color="orange"
                                        size="md"
                                    >
                                        Cambiar contraseña
                                    </Button>
                                </Stack>
                            </form>
                        </Paper>
                    </div>
                </div>
            </div>
        );
    }

    // ── Normal login form ────────────────────────────────────────────────────
    return (
        <div className={classes.wrapper}>
            {/* Left: Hero / Branding */}
            {HeroPanel}

            {/* Right: Login form */}
            <div className={classes.formSide}>
                <div className={`${classes.formContainer} ${classes.fadeInDelay}`}>
                    {/* Mobile branding — shown on small screens only */}
                    <div className={classes.mobileBranding}>
                        <div className={classes.mobileLogo}>
                            Blend<span className={classes.mobileLogoAccent}>POS</span>
                        </div>
                        <span className={classes.mobileTagline}>POS moderno para negocios reales</span>
                    </div>

                    <Paper p="xl" className={classes.formCard}>
                        <h2 className={classes.formTitle}>Iniciar sesión</h2>
                        <p className={classes.formSubtitle}>
                            Accedé a tu sistema y comenzá a operar
                        </p>

                        {error && (
                            <Alert icon={<AlertCircle size={16} />} color="red" mb="md" variant="light">
                                {error}
                            </Alert>
                        )}

                        <form onSubmit={handleSubmit}>
                            <Stack gap="md">
                                <TextInput
                                    id="login-email"
                                    label="Correo electrónico"
                                    placeholder="ejemplo@negocio.com"
                                    autoComplete="username"
                                    {...form.getInputProps('email')}
                                    data-autofocus
                                />
                                <PasswordInput
                                    id="login-password"
                                    label="Contraseña"
                                    placeholder="••••••••"
                                    autoComplete="current-password"
                                    {...form.getInputProps('password')}
                                />

                                <Group justify="space-between" align="center" mt={2}>
                                    <Checkbox
                                        id="login-remember"
                                        checked={rememberMe}
                                        onChange={(event) => setRememberMe(event.currentTarget.checked)}
                                        label="Recordarme"
                                        size="sm"
                                    />
                                    <Anchor
                                        href="#"
                                        className={classes.forgotLink}
                                        onClick={(event) => event.preventDefault()}
                                    >
                                        ¿Olvidaste tu contraseña?
                                    </Anchor>
                                </Group>

                                <Button
                                    id="login-submit"
                                    type="submit"
                                    fullWidth
                                    loading={loading}
                                    disabled={loading}
                                    mt="xs"
                                    size="md"
                                >
                                    {loading ? 'Ingresando...' : 'Ingresar'}
                                </Button>

                                {/* Divider */}
                                <div className={classes.orDivider}>
                                    <div className={classes.orDividerLine} />
                                    <span className={classes.orDividerText}>o continuá con</span>
                                    <div className={classes.orDividerLine} />
                                </div>

                                <Button
                                    type="button"
                                    fullWidth
                                    variant="default"
                                    size="md"
                                    leftSection={<Chrome size={16} />}
                                >
                                    Continuar con Google
                                </Button>
                            </Stack>
                        </form>

                        {/* Footer */}
                        <div className={classes.formFooter}>
                            <Text size="xs" c="dimmed" ta="center" style={{ fontFamily: 'Manrope, sans-serif' }}>
                                ¿No tenés cuenta?{' '}
                                <Anchor component={Link} to="/register" size="xs" fw={600} style={{ color: '#60a5fa' }}>
                                    Creá tu negocio gratis
                                </Anchor>
                            </Text>

                            <div className={classes.footerLinks}>
                                <span>© 2026 BlendPOS</span>
                                <div className={classes.footerDivider} />
                                <button
                                    type="button"
                                    className={classes.footerLink}
                                    onClick={() => setSupportOpen(true)}
                                >
                                    Soporte técnico
                                </button>
                                <div className={classes.footerDivider} />
                                <button
                                    type="button"
                                    className={classes.footerLink}
                                    onClick={() => setTermsOpen(true)}
                                >
                                    Términos y condiciones
                                </button>
                            </div>
                        </div>

                        {/* Modals */}
                        <SupportModal opened={supportOpen} onClose={() => setSupportOpen(false)} />
                        <TermsModal opened={termsOpen} onClose={() => setTermsOpen(false)} />
                    </Paper>
                </div>
            </div>
        </div>
    );
}
