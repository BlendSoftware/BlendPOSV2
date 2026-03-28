import { Modal, Text, Anchor, Stack, Group, Badge, ThemeIcon, Divider } from '@mantine/core';
import {
    MessageCircle, Mail, Clock, Zap, Book,
    HeadphonesIcon, ExternalLink, Phone,
} from 'lucide-react';
import classes from './SupportModal.module.css';

interface SupportModalProps {
    opened: boolean;
    onClose: () => void;
}

const CHANNELS = [
    {
        icon: <MessageCircle size={18} />,
        title: 'Chat en vivo',
        description: 'Respuesta en menos de 5 minutos en horario hábil.',
        badge: 'En línea',
        badgeColor: 'green',
        action: 'Iniciar chat',
        href: 'https://wa.me/+5491100000000',
    },
    {
        icon: <Mail size={18} />,
        title: 'Email',
        description: 'soporte@blendpos.com.ar — respondemos en menos de 24hs.',
        badge: '24hs',
        badgeColor: 'blue',
        action: 'Enviar email',
        href: 'mailto:soporte@blendpos.com.ar',
    },
    {
        icon: <Phone size={18} />,
        title: 'Teléfono',
        description: 'Lun–Vie de 9:00 a 18:00 hs (Argentina).',
        badge: 'Lun–Vie',
        badgeColor: 'gray',
        action: 'Llamar',
        href: 'tel:+5491100000000',
    },
];

const FAQ = [
    {
        q: '¿Cómo funciona el modo offline?',
        a: 'BlendPOS almacena los datos localmente y sincroniza automáticamente cuando recupera la conexión. Hasta 48 hs de autonomía.',
    },
    {
        q: '¿Puedo usar BlendPOS en más de una computadora?',
        a: 'Sí, cada terminal se conecta a tu cuenta. Podés tener múltiples puntos de venta activos al mismo tiempo.',
    },
    {
        q: '¿Cómo actualizo los precios de mis productos?',
        a: 'Desde el Panel Admin → Productos → editá el precio. El cambio se replica a todos tus terminales en tiempo real.',
    },
    {
        q: '¿BlendPOS emite facturas electrónicas?',
        a: 'Sí. Integración nativa con AFIP para tickets, facturas A y B. Requiere CUIT y credenciales AFIP configuradas.',
    },
];

export function SupportModal({ opened, onClose }: SupportModalProps) {
    return (
        <Modal
            opened={opened}
            onClose={onClose}
            title={
                <Group gap={8}>
                    <ThemeIcon size={28} radius={8} className={classes.titleIcon}>
                        <HeadphonesIcon size={15} />
                    </ThemeIcon>
                    <Text fw={700} size="sm" ff="Manrope, sans-serif">Soporte Técnico</Text>
                </Group>
            }
            size="lg"
            centered
            radius="md"
        >
            <Stack gap={20} className={classes.body}>

                {/* Channels */}
                <div>
                    <Text className={classes.sectionTitle}>Canales de atención</Text>
                    <Stack gap={8} mt={10}>
                        {CHANNELS.map((ch) => (
                            <div key={ch.title} className={classes.channelCard}>
                                <div className={classes.channelIconWrap}>
                                    {ch.icon}
                                </div>
                                <div className={classes.channelInfo}>
                                    <Group gap={8} align="center">
                                        <Text className={classes.channelTitle}>{ch.title}</Text>
                                        <Badge
                                            size="xs"
                                            color={ch.badgeColor}
                                            variant="light"
                                            className={classes.channelBadge}
                                        >
                                            {ch.badge}
                                        </Badge>
                                    </Group>
                                    <Text className={classes.channelDesc}>{ch.description}</Text>
                                </div>
                                <Anchor
                                    href={ch.href}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className={classes.channelAction}
                                >
                                    {ch.action}
                                    <ExternalLink size={11} />
                                </Anchor>
                            </div>
                        ))}
                    </Stack>
                </div>

                <Divider className={classes.divider} />

                {/* Status */}
                <div className={classes.statusBanner}>
                    <Zap size={14} className={classes.statusIcon} />
                    <Text className={classes.statusText}>
                        Todos los sistemas operativos — Tiempo de actividad: 99.9%
                    </Text>
                </div>

                <Divider className={classes.divider} />

                {/* FAQ */}
                <div>
                    <Group gap={6} mb={10}>
                        <Book size={14} className={classes.faqIcon} />
                        <Text className={classes.sectionTitle}>Preguntas frecuentes</Text>
                    </Group>
                    <Stack gap={6}>
                        {FAQ.map((item) => (
                            <div key={item.q} className={classes.faqItem}>
                                <Text className={classes.faqQ}>{item.q}</Text>
                                <Text className={classes.faqA}>{item.a}</Text>
                            </div>
                        ))}
                    </Stack>
                </div>

                {/* Footer hint */}
                <Text className={classes.footerNote}>
                    BlendPOS · Argentina · <Anchor href="https://blendpos.com.ar" target="_blank" className={classes.footerLink}>blendpos.com.ar</Anchor>
                </Text>
            </Stack>
        </Modal>
    );
}
