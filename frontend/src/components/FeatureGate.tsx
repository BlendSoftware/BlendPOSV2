import type { ReactNode } from 'react';
import { Card, Text, Button, Stack, Center, Loader, ThemeIcon } from '@mantine/core';
import { Lock, Sparkles } from 'lucide-react';
import { useFeatureGate } from '../hooks/useFeatureGate';
import type { Feature } from '../config/plans';

interface FeatureGateProps {
    /** Feature flag key — must match the backend plan features JSONB keys */
    feature: Feature;
    /** Content to render when the feature is enabled */
    children: ReactNode;
    /** Optional custom fallback — defaults to an upgrade banner */
    fallback?: ReactNode;
}

function DefaultUpgradeBanner({ feature, planRequired }: { feature: Feature; planRequired: string }) {
    const featureLabel = feature.replace(/_/g, ' ');
    return (
        <Center py="xl">
            <Card shadow="sm" padding="lg" radius="md" withBorder maw={420} w="100%">
                <Stack align="center" gap="md">
                    <ThemeIcon size="xl" radius="xl" variant="light" color="blue">
                        <Lock size={24} />
                    </ThemeIcon>
                    <Text fw={600} size="lg" ta="center">
                        Disponible en plan {planRequired}
                    </Text>
                    <Text c="dimmed" size="sm" ta="center">
                        La funcionalidad de {featureLabel} no esta incluida en tu plan actual.
                        Mejora a <Text span fw={600} c="blue">{planRequired}</Text> para desbloquearla.
                    </Text>
                    <Button
                        component="a"
                        href="/admin/dashboard"
                        variant="filled"
                        color="blue"
                        radius="md"
                        leftSection={<Sparkles size={16} />}
                    >
                        Ver planes
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
 * <FeatureGate feature="ai_assistant">
 *     <AIAssistant />
 * </FeatureGate>
 */
export function FeatureGate({ feature, children, fallback }: FeatureGateProps) {
    const { allowed, loading, planRequired } = useFeatureGate(feature);

    if (loading) {
        return <Center py="xl"><Loader size="sm" /></Center>;
    }

    if (!allowed) {
        return <>{fallback ?? <DefaultUpgradeBanner feature={feature} planRequired={planRequired} />}</>;
    }

    return <>{children}</>;
}
