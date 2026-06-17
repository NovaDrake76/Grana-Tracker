ALTER TABLE investments DROP CONSTRAINT IF EXISTS investments_currency_check;
ALTER TABLE investments DROP COLUMN IF EXISTS currency;
ALTER TABLE investments DROP COLUMN IF EXISTS purchase_price;
