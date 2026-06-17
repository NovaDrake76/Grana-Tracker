ALTER TABLE investments ADD COLUMN IF NOT EXISTS purchase_price DECIMAL(18,8);
ALTER TABLE investments ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'BRL';
-- Constraint cannot be added with IF NOT EXISTS in older Postgres, but the
-- migration runner is idempotent (tracks schema_migrations), so plain ADD
-- CONSTRAINT is fine. The DO block swallows duplicate_object for paranoia.
DO $$ BEGIN
  ALTER TABLE investments ADD CONSTRAINT investments_currency_check CHECK (currency IN ('BRL','USD'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
-- Best-effort backfill: if quantity is known and > 0, derive
-- purchase_price = amount_invested / quantity so existing rows are not stranded
-- with NULL purchase_price after the migration.
UPDATE investments
   SET purchase_price = amount_invested / quantity
 WHERE quantity IS NOT NULL AND quantity > 0 AND purchase_price IS NULL;
