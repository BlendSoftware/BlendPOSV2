-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 000028: Subscriptions table for billing (MercadoPago integration)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE subscriptions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    plan_id               UUID NOT NULL REFERENCES plans(id),
    mp_subscription_id    VARCHAR(255),       -- MercadoPago subscription/preapproval ID
    mp_payer_id           VARCHAR(255),       -- MercadoPago payer ID
    status                VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending, active, paused, cancelled
    current_period_start  TIMESTAMPTZ,
    current_period_end    TIMESTAMPTZ,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_mp_id  ON subscriptions(mp_subscription_id);

-- RLS: subscriptions follow the same tenant isolation as all other tables.
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY subscriptions_tenant_isolation ON subscriptions
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);
