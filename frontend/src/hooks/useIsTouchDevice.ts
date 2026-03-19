import { useMediaQuery } from '@mantine/hooks';

interface TouchDeviceInfo {
    /** True if the device has touch capability */
    isTouch: boolean;
    /** True if screen width is between 768px and 1024px (tablet range) */
    isTablet: boolean;
}

/**
 * Detects touch capability and tablet screen size.
 * Uses Mantine's useMediaQuery internally for SSR-safe media queries.
 */
export function useIsTouchDevice(): TouchDeviceInfo {
    const isCoarse = useMediaQuery('(pointer: coarse)');
    const isTabletWidth = useMediaQuery('(min-width: 768px) and (max-width: 1024px)');

    return {
        isTouch: isCoarse ?? false,
        isTablet: isTabletWidth ?? false,
    };
}
