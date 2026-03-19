import type { ReactNode } from 'react';
import { Card, Text, Button, Stack, Center, Loader, ThemeIcon } from '@mantine/core';
import { Lock } from 'lucide-react';
import { useFeature } from '../hooks/useFeature';

interface FeatureGateProps {
    /** Feature flag key — must match the backend plan features JSONB keys */
    feature: string;
    /** Content to render when the feature is enabled */
    children: ReactNode;
    /** Optional custom fallback — defaults to an upgrade banner */
    fallback?: ReactNode;
}

function DefaultUpgradeBanner({ feature }: { feature: string }) {
    return (
        <Center py="xl">
            <Card shadow="sm" padding="lg" radius="md" withBorder maw={420} w="100%">
                <Stack align="center" gap="md">
                    <ThemeIcon size="xl" radius="xl" variant="light" color="gray">
                        <Lock size={24} />
                    </ThemeIcon>
                    <Text fw={600} size="lg" ta="center">
                        Funcionalidad no disponible
                    </Text>
                    <Text c="dimmed" size="sm" ta="center">
                        Tu plan actual no incluye esta funcionalidad ({feature.replace(/_/g, ' ')}).
                        Actualiza tu plan para desbloquearla.
                    </Text>
                    <Button
                        component="a"
                        href="/admin/dashboard"
                        variant="filled"
                        color="blue"
                        radius="md"
                    >
                        Ver planes disponibles
                    </Button>
                </Stack>
            </Card>
        </Center>
    );
}

/**
 * FeatureGate — conditionally renders children based on plan feature flags.
 *
 * Features that are not enabled are shown as locked with an upgrade CTA,
 * NOT hidden — this drives upsell awareness.
 *
 * @example
 * <FeatureGate feature="analytics_avanzados">
 *     <AdvancedAnalytics />
 * </FeatureGate>
 */
export function FeatureGate({ feature, children, fallback }: FeatureGateProps) {
    const { enabled, loading } = useFeature(feature);

    if (loading) {
        return <Center py="xl"><Loader size="sm" /></Center>;
    }

    if (!enabled) {
        return <>{fallback ?? <DefaultUpgradeBanner feature={feature} />}</>;
    }

    return <>{children}</>;
}
