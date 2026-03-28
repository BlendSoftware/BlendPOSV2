import { Stack, Title, Text, SimpleGrid, Card, Group, Badge, Box } from '@mantine/core';
import { Check, Palette, Monitor, Type } from 'lucide-react';
import { POS_THEME_PRESETS } from '../../theme/posThemes';
import type { PosTheme } from '../../theme/posThemes';
import { usePosThemeStore } from '../../store/usePosThemeStore';
import { notifications } from '@mantine/notifications';

// ── Mini POS mockup for preview ─────────────────────────────────────────────

function ThemePreview({ theme }: { theme: PosTheme }) {
    const { colors, borderRadius, font } = theme;
    const radius = parseInt(borderRadius) || 8;
    const miniRadius = Math.max(2, radius / 2);

    return (
        <Box
            style={{
                background: colors.background,
                borderRadius: miniRadius + 2,
                padding: 8,
                fontFamily: font.family,
                overflow: 'hidden',
                border: `1px solid ${colors.border}`,
                height: 130,
                display: 'flex',
                flexDirection: 'column',
                gap: 4,
            }}
        >
            {/* Header mockup */}
            <Box
                style={{
                    background: colors.surface,
                    borderRadius: miniRadius,
                    padding: '4px 8px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    borderBottom: `1px solid ${colors.border}`,
                }}
            >
                <Text size="9px" fw={800} style={{ color: colors.text, fontFamily: font.heading }}>
                    BlendPOS
                </Text>
                <Box
                    style={{
                        width: 6,
                        height: 6,
                        borderRadius: '50%',
                        background: colors.primary,
                    }}
                />
            </Box>

            {/* Main area mockup */}
            <Box style={{ display: 'flex', gap: 4, flex: 1, minHeight: 0 }}>
                {/* Table mockup */}
                <Box
                    style={{
                        flex: 1,
                        background: colors.surface,
                        borderRadius: miniRadius,
                        padding: 6,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 3,
                    }}
                >
                    {[1, 2, 3].map((i) => (
                        <Box
                            key={i}
                            style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                                alignItems: 'center',
                                padding: '2px 4px',
                                borderRadius: 2,
                                background: i === 1
                                    ? `${colors.primary}22`
                                    : 'transparent',
                                borderLeft: i === 1
                                    ? `2px solid ${colors.primary}`
                                    : '2px solid transparent',
                            }}
                        >
                            <Box
                                style={{
                                    width: 28 + i * 6,
                                    height: 4,
                                    borderRadius: 2,
                                    background: colors.text,
                                    opacity: 0.4,
                                }}
                            />
                            <Box
                                style={{
                                    width: 18,
                                    height: 4,
                                    borderRadius: 2,
                                    background: colors.success,
                                    opacity: 0.7,
                                }}
                            />
                        </Box>
                    ))}
                </Box>

                {/* Total panel mockup */}
                <Box
                    style={{
                        width: 56,
                        background: colors.surface,
                        borderRadius: miniRadius,
                        padding: 6,
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                    }}
                >
                    <Text size="11px" fw={800} style={{ color: colors.text, fontFamily: font.mono }}>
                        $99
                    </Text>
                    <Box
                        style={{
                            width: '100%',
                            height: 12,
                            borderRadius: miniRadius,
                            background: colors.success,
                        }}
                    />
                </Box>
            </Box>

            {/* Footer mockup */}
            <Box
                style={{
                    background: colors.surface,
                    borderRadius: miniRadius,
                    padding: '2px 6px',
                    display: 'flex',
                    gap: 4,
                    alignItems: 'center',
                }}
            >
                {['F2', 'F5', 'F10'].map((key) => (
                    <Box
                        key={key}
                        style={{
                            fontSize: 6,
                            fontWeight: 700,
                            padding: '1px 3px',
                            borderRadius: 2,
                            background: `${colors.primary}22`,
                            color: colors.primary,
                            fontFamily: font.mono,
                        }}
                    >
                        {key}
                    </Box>
                ))}
            </Box>
        </Box>
    );
}

// ── Color swatches ──────────────────────────────────────────────────────────

function ColorSwatches({ theme }: { theme: PosTheme }) {
    const swatches = [
        { color: theme.colors.primary, label: 'Primario' },
        { color: theme.colors.background, label: 'Fondo' },
        { color: theme.colors.surface, label: 'Panel' },
        { color: theme.colors.success, label: 'Exito' },
        { color: theme.colors.danger, label: 'Error' },
    ];

    return (
        <Group gap={4} mt={6}>
            {swatches.map((s) => (
                <Box
                    key={s.label}
                    title={s.label}
                    style={{
                        width: 16,
                        height: 16,
                        borderRadius: 4,
                        background: s.color,
                        border: '1px solid rgba(128,128,128,0.3)',
                    }}
                />
            ))}
        </Group>
    );
}

// ── Theme Card ──────────────────────────────────────────────────────────────

function ThemeCard({ theme, isActive, onSelect }: {
    theme: PosTheme;
    isActive: boolean;
    onSelect: () => void;
}) {
    const styleBadgeColor = theme.style === 'sharp' ? 'gray' : theme.style === 'rounded' ? 'blue' : 'grape';

    return (
        <Card
            shadow={isActive ? 'lg' : 'sm'}
            padding="md"
            radius="md"
            withBorder
            onClick={onSelect}
            style={{
                cursor: 'pointer',
                borderColor: isActive ? 'var(--mantine-color-blue-6)' : undefined,
                borderWidth: isActive ? 2 : 1,
                transition: 'all 150ms ease',
                position: 'relative',
                overflow: 'visible',
            }}
        >
            {isActive && (
                <Box
                    style={{
                        position: 'absolute',
                        top: -8,
                        right: -8,
                        width: 24,
                        height: 24,
                        borderRadius: '50%',
                        background: 'var(--mantine-color-blue-6)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        zIndex: 1,
                    }}
                >
                    <Check size={14} color="white" />
                </Box>
            )}

            <ThemePreview theme={theme} />

            <Group justify="space-between" mt="sm" gap="xs">
                <Text fw={700} size="sm">{theme.name}</Text>
                <Badge size="xs" variant="light" color={styleBadgeColor}>
                    {theme.style}
                </Badge>
            </Group>

            <Text size="xs" c="dimmed" mt={4} lineClamp={2}>
                {theme.description}
            </Text>

            <Group gap={6} mt={8}>
                <Group gap={4}>
                    <Type size={11} style={{ opacity: 0.5 }} />
                    <Text size="10px" c="dimmed" style={{ fontFamily: theme.font.family }}>
                        {theme.font.family.split("'")[1] || 'System'}
                    </Text>
                </Group>
            </Group>

            <ColorSwatches theme={theme} />
        </Card>
    );
}

// ── Page Component ──────────────────────────────────────────────────────────

export function AparienciaPage() {
    const { activeThemeId, setTheme } = usePosThemeStore();

    const handleSelect = (themeId: string) => {
        setTheme(themeId);
        const theme = POS_THEME_PRESETS.find((t) => t.id === themeId);
        if (theme) {
            notifications.show({
                title: 'Tema aplicado',
                message: `Se aplico el tema "${theme.name}" al terminal POS.`,
                color: 'blue',
                icon: <Palette size={16} />,
                autoClose: 3000,
            });
        }
    };

    return (
        <Stack gap="lg" p="md">
            <div>
                <Group gap="sm" mb={4}>
                    <Monitor size={22} />
                    <Title order={3}>Apariencia del POS</Title>
                </Group>
                <Text c="dimmed" size="sm">
                    Personaliza los colores, tipografia y estilo visual del terminal de ventas.
                    Los cambios se aplican de forma instantanea.
                </Text>
            </div>

            <SimpleGrid cols={{ base: 1, xs: 2, md: 3 }} spacing="md">
                {POS_THEME_PRESETS.map((theme) => (
                    <ThemeCard
                        key={theme.id}
                        theme={theme}
                        isActive={activeThemeId === theme.id}
                        onSelect={() => handleSelect(theme.id)}
                    />
                ))}
            </SimpleGrid>
        </Stack>
    );
}
