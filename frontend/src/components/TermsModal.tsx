import { Modal, Text, Anchor, Stack, ThemeIcon, Divider, Group } from '@mantine/core';
import { Shield, ScrollText, AlertTriangle } from 'lucide-react';
import classes from './TermsModal.module.css';

interface TermsModalProps {
    opened: boolean;
    onClose: () => void;
}

const SECTIONS = [
    {
        id: '1',
        title: '1. Aceptación de los Términos',
        content: `Al acceder y utilizar BlendPOS, el Usuario acepta quedar vinculado por estos Términos y Condiciones. Si no está de acuerdo con alguna parte de estos términos, no podrá utilizar el servicio.`,
    },
    {
        id: '2',
        title: '2. Descripción del Servicio',
        content: `BlendPOS es una plataforma SaaS de gestión de punto de venta (POS) orientada a comercios en Argentina. El servicio incluye: terminal de ventas, gestión de productos e inventario, reportes, integración con AFIP para facturación electrónica, y soporte multi-sucursal.`,
    },
    {
        id: '3',
        title: '3. Uso Aceptable',
        content: `El Usuario se compromete a: (a) utilizar el servicio únicamente con fines legales y conforme a la normativa argentina vigente; (b) no intentar acceder a cuentas de otros usuarios; (c) no reproducir, duplicar, copiar o revender ninguna parte del servicio sin consentimiento expreso.`,
    },
    {
        id: '4',
        title: '4. Protección de Datos Personales',
        content: `BlendPOS cumple con la Ley 25.326 de Protección de Datos Personales (Argentina). Los datos ingresados por el Usuario son tratados de forma confidencial y no son cedidos a terceros salvo requerimiento legal. El Usuario puede solicitar acceso, rectificación o eliminación de sus datos en cualquier momento.`,
    },
    {
        id: '5',
        title: '5. Facturación y Pagos',
        content: `El servicio se factura en pesos argentinos (ARS) en modalidad de suscripción mensual. Los precios pueden ajustarse con un preaviso de 30 días. El impago de la suscripción puede resultar en la suspensión del acceso, sin eliminar los datos almacenados.`,
    },
    {
        id: '6',
        title: '6. Disponibilidad del Servicio',
        content: `BlendPOS se esfuerza por mantener una disponibilidad del 99.9% mensual. Pueden producirse interrupciones planificadas para mantenimiento, notificadas con anticipación. El servicio opera en modo offline, por lo que interrupciones en la conectividad no afectan la operatoria local.`,
    },
    {
        id: '7',
        title: '7. Limitación de Responsabilidad',
        content: `BlendPOS no será responsable por pérdidas indirectas, incidentales o consecuentes derivadas del uso del servicio. La responsabilidad máxima total de BlendPOS no excederá el monto abonado por el Usuario en los últimos 3 meses.`,
    },
    {
        id: '8',
        title: '8. Modificaciones',
        content: `BlendPOS se reserva el derecho de modificar estos Términos en cualquier momento. Los cambios materiales se notificarán con al menos 15 días de anticipación mediante email. El uso continuado del servicio tras la notificación implica la aceptación de los nuevos términos.`,
    },
    {
        id: '9',
        title: '9. Ley Aplicable y Jurisdicción',
        content: `Estos Términos se rigen por las leyes de la República Argentina. Cualquier disputa será sometida a la jurisdicción de los tribunales ordinarios de la Ciudad Autónoma de Buenos Aires.`,
    },
];

export function TermsModal({ opened, onClose }: TermsModalProps) {
    return (
        <Modal
            opened={opened}
            onClose={onClose}
            title={
                <Group gap={8}>
                    <ThemeIcon size={28} radius={8} className={classes.titleIcon}>
                        <ScrollText size={15} />
                    </ThemeIcon>
                    <Text fw={700} size="sm" ff="Manrope, sans-serif">Términos y Condiciones</Text>
                </Group>
            }
            size="xl"
            centered
            radius="md"
        >
            <Stack gap={0} className={classes.body}>

                {/* Header disclaimer */}
                <div className={classes.disclaimer}>
                    <AlertTriangle size={14} className={classes.disclaimerIcon} />
                    <Text className={classes.disclaimerText}>
                        Última actualización: marzo 2026 · Versión 1.0
                    </Text>
                </div>

                <Text className={classes.intro}>
                    Los siguientes Términos y Condiciones regulan el acceso y uso de <strong>BlendPOS</strong>, 
                    software de punto de venta desarrollado y operado en la República Argentina. 
                    Por favor léalos detenidamente antes de utilizar el servicio.
                </Text>

                <Divider className={classes.divider} my="md" />

                {/* Sections */}
                <Stack gap={4}>
                    {SECTIONS.map((sec, idx) => (
                        <div key={sec.id} className={classes.section}>
                            <Text className={classes.sectionTitle}>{sec.title}</Text>
                            <Text className={classes.sectionContent}>{sec.content}</Text>
                            {idx < SECTIONS.length - 1 && (
                                <Divider className={classes.sectionDivider} mt={14} />
                            )}
                        </div>
                    ))}
                </Stack>

                <Divider className={classes.divider} my="md" />

                {/* Footer */}
                <div className={classes.footer}>
                    <Shield size={14} className={classes.footerIcon} />
                    <Text className={classes.footerText}>
                        ¿Tenés preguntas sobre estos términos?{' '}
                        <Anchor href="mailto:legal@blendpos.com.ar" className={classes.footerLink}>
                            legal@blendpos.com.ar
                        </Anchor>
                    </Text>
                </div>

            </Stack>
        </Modal>
    );
}
