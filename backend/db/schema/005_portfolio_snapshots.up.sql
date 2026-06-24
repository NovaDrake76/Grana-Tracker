CREATE TABLE IF NOT EXISTS portfolio_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    total_value DECIMAL(18,8) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'BRL',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(portfolio_id, snapshot_date)
);
CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_portfolio_date
    ON portfolio_snapshots(portfolio_id, snapshot_date DESC);
