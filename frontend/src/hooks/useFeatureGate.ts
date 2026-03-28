// ─────────────────────────────────────────────────────────────────────────────
// useFeatureGate — Plan-based feature gating hook
//
// Reads the tenant's plan from the auth store and checks whether a given
// Feature is available. Returns both the allowed state and the name of the
// plan that would unlock it.
// ─────────────────────────────────────────────────────────────────────────────

import { useMemo } from 'react';
import { useFeature } from './useFeature';
import {
    type Feature,
    getMinimumPlanForFeature,
} from '../config/plans';

interface FeatureGateResult {
    /** Whether the current plan includes this feature. */
    allowed: boolean;
    /** True while the plan is still being loaded from the API. */
    loading: boolean;
    /** Human-readable name of the plan that unlocks this feature. */
    planRequired: string;
}

/**
 * useFeatureGate — checks if a feature is available in the tenant's current plan.
 *
 * Uses the existing useFeature hook (which fetches plan features from the backend)
 * and enriches it with the minimum plan info for upgrade prompts.
 *
 * @example
 * const { allowed, planRequired } = useFeatureGate('ai_assistant');
 * if (!allowed) show upgrade prompt for planRequired
 */
export function useFeatureGate(feature: Feature): FeatureGateResult {
    const { enabled, loading } = useFeature(feature);

    const planRequired = useMemo(
        () => getMinimumPlanForFeature(feature).name,
        [feature],
    );

    return { allowed: enabled, loading, planRequired };
}

/**
 * useCanAccess — shorthand that returns a simple boolean.
 */
export function useCanAccess(feature: Feature): boolean {
    const { allowed } = useFeatureGate(feature);
    return allowed;
}
