import { Group, Text, Kbd, Flex } from '@mantine/core';
import styles from './HotkeysFooter.module.css';

interface HotkeyHint {
    key: string;
    label: string;
}

interface HotkeysFooterProps {
    variant?: 'footer' | 'rail';
}

const HOTKEYS: HotkeyHint[] = [
    { key: 'F2', label: 'Buscar' },
    { key: 'F3', label: 'Desc. ítem' },
    { key: 'F5', label: 'Consultar $ ' },
    { key: 'F7', label: 'Historial' },
    { key: 'F8', label: 'Desc. global' },
    { key: 'F10', label: 'Cobrar' },
    { key: 'ESC', label: 'Cancelar' },
];

export function HotkeysFooter({ variant = 'footer' }: HotkeysFooterProps) {
    return (
        <footer className={`${styles.footer} ${variant === 'rail' ? styles.footerRail : ''}`}>
            <Flex
                align="center"
                justify="space-between"
                h="100%"
                px="md"
                className={`${styles.hotkeysRail} ${variant === 'rail' ? styles.hotkeysRailVertical : ''}`}
            >
                {HOTKEYS.map((hotkey, index) => (
                    <Group key={hotkey.key} gap="xs" className={`${styles.hotkeyItem} ${variant === 'rail' ? styles.hotkeyItemVertical : ''}`}>
                        <Kbd className={styles.kbd}>{hotkey.key}</Kbd>
                        <Text size="xs" className={styles.hotkeyLabel}>
                            {hotkey.label}
                        </Text>
                        {index < HOTKEYS.length - 1 && <div className={`${styles.hotkeyDivider} ${variant === 'rail' ? styles.hotkeyDividerVertical : ''}`} />}
                    </Group>
                ))}

                {variant !== 'rail' && (
                    <Text size="xs" className={styles.opsHint}>
                        Navegacion rapida: ↑↓ seleccionar | +/- cantidad | Supr eliminar
                    </Text>
                )}
            </Flex>
        </footer>
    );
}
