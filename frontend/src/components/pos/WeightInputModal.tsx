import { useState, useRef, useEffect } from 'react';
import { Modal, NumberInput, Button, Group, Text, Stack, Badge } from '@mantine/core';
import { Scale } from 'lucide-react';

interface WeightInputModalProps {
    opened: boolean;
    onClose: () => void;
    onConfirm: (peso: number) => void;
    productName: string;
    precioUnitario: number;
    unidadMedida: 'kg' | 'gramo';
}

function formatCurrency(value: number): string {
    return new Intl.NumberFormat('es-AR', {
        style: 'currency',
        currency: 'ARS',
        minimumFractionDigits: 2,
    }).format(value);
}

export function WeightInputModal({
    opened,
    onClose,
    onConfirm,
    productName,
    precioUnitario,
    unidadMedida,
}: WeightInputModalProps) {
    const [peso, setPeso] = useState<number | string>(0);
    const inputRef = useRef<HTMLInputElement>(null);

    // Auto-focus and reset when modal opens
    useEffect(() => {
        if (opened) {
            // Small delay so the modal renders before we focus and reset
            setTimeout(() => {
                setPeso(0);
                inputRef.current?.select();
            }, 50);
        }
    }, [opened]);

    const numericPeso = typeof peso === 'string' ? parseFloat(peso) : peso;
    const isValid = !isNaN(numericPeso) && numericPeso > 0;
    const subtotal = isValid ? precioUnitario * numericPeso : 0;
    const suffix = unidadMedida === 'kg' ? 'kg' : 'g';
    const step = unidadMedida === 'kg' ? 0.001 : 1;
    const decimalScale = unidadMedida === 'kg' ? 3 : 0;

    const handleConfirm = () => {
        if (!isValid) return;
        onConfirm(numericPeso);
        onClose();
    };

    return (
        <Modal
            opened={opened}
            onClose={onClose}
            title={
                <Group gap="xs">
                    <Scale size={18} />
                    <Text fw={700}>Ingresar peso</Text>
                </Group>
            }
            size="sm"
            centered
            trapFocus
        >
            <Stack gap="md">
                <Group justify="space-between">
                    <Text size="sm" fw={600}>{productName}</Text>
                    <Badge variant="light" color="blue" size="lg">
                        {formatCurrency(precioUnitario)}/{suffix}
                    </Badge>
                </Group>

                <NumberInput
                    ref={inputRef}
                    label={`Peso (${suffix})`}
                    placeholder={`Ej: ${unidadMedida === 'kg' ? '1.250' : '500'}`}
                    value={peso}
                    onChange={setPeso}
                    min={0.001}
                    step={step}
                    decimalScale={decimalScale}
                    decimalSeparator=","
                    suffix={` ${suffix}`}
                    size="lg"
                    autoFocus
                    data-pos-focusable
                    onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                            e.preventDefault();
                            handleConfirm();
                        }
                        if (e.key === 'Escape') {
                            e.preventDefault();
                            onClose();
                        }
                    }}
                />

                {isValid && (
                    <Group justify="space-between">
                        <Text size="sm" c="dimmed">Subtotal:</Text>
                        <Text size="lg" fw={700} c="teal">
                            {formatCurrency(subtotal)}
                        </Text>
                    </Group>
                )}

                <Group justify="flex-end" mt="xs">
                    <Button variant="subtle" onClick={onClose}>
                        Cancelar
                    </Button>
                    <Button
                        onClick={handleConfirm}
                        disabled={!isValid}
                        leftSection={<Scale size={16} />}
                    >
                        Confirmar peso
                    </Button>
                </Group>
            </Stack>
        </Modal>
    );
}
