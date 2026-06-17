CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker TEXT NOT NULL,
    name TEXT NOT NULL,
    asset_type TEXT NOT NULL CHECK (asset_type IN ('stock','crypto','etf','index')),
    source TEXT NOT NULL,
    external_id TEXT,
    currency TEXT NOT NULL DEFAULT 'USD',
    market TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ticker, asset_type)
);

CREATE INDEX IF NOT EXISTS idx_assets_search ON assets USING gin (to_tsvector('simple', ticker || ' ' || name));
CREATE INDEX IF NOT EXISTS idx_assets_ticker_lower ON assets (LOWER(ticker));

-- Seed B3 stocks (BRL, BVMF).
INSERT INTO assets (ticker, name, asset_type, source, external_id, currency, market) VALUES
    ('VALE3', 'VALE3 - Vale S.A.', 'stock', 'b3', 'VALE3', 'BRL', 'BVMF'),
    ('PETR4', 'PETR4 - Petrobras PN', 'stock', 'b3', 'PETR4', 'BRL', 'BVMF'),
    ('ITUB4', 'ITUB4 - Itaú Unibanco PN', 'stock', 'b3', 'ITUB4', 'BRL', 'BVMF'),
    ('BBDC4', 'BBDC4 - Bradesco PN', 'stock', 'b3', 'BBDC4', 'BRL', 'BVMF'),
    ('ABEV3', 'ABEV3 - Ambev S.A.', 'stock', 'b3', 'ABEV3', 'BRL', 'BVMF'),
    ('MGLU3', 'MGLU3 - Magazine Luiza', 'stock', 'b3', 'MGLU3', 'BRL', 'BVMF'),
    ('WEGE3', 'WEGE3 - WEG S.A.', 'stock', 'b3', 'WEGE3', 'BRL', 'BVMF'),
    ('B3SA3', 'B3SA3 - B3 S.A.', 'stock', 'b3', 'B3SA3', 'BRL', 'BVMF'),
    ('BBAS3', 'BBAS3 - Banco do Brasil', 'stock', 'b3', 'BBAS3', 'BRL', 'BVMF'),
    ('RENT3', 'RENT3 - Localiza', 'stock', 'b3', 'RENT3', 'BRL', 'BVMF'),
    ('RADL3', 'RADL3 - Raia Drogasil', 'stock', 'b3', 'RADL3', 'BRL', 'BVMF'),
    ('LREN3', 'LREN3 - Lojas Renner', 'stock', 'b3', 'LREN3', 'BRL', 'BVMF'),
    ('JBSS3', 'JBSS3 - JBS', 'stock', 'b3', 'JBSS3', 'BRL', 'BVMF'),
    ('SUZB3', 'SUZB3 - Suzano', 'stock', 'b3', 'SUZB3', 'BRL', 'BVMF'),
    ('ELET3', 'ELET3 - Eletrobras ON', 'stock', 'b3', 'ELET3', 'BRL', 'BVMF'),
    ('ELET6', 'ELET6 - Eletrobras PNB', 'stock', 'b3', 'ELET6', 'BRL', 'BVMF'),
    ('GGBR4', 'GGBR4 - Gerdau PN', 'stock', 'b3', 'GGBR4', 'BRL', 'BVMF'),
    ('USIM5', 'USIM5 - Usiminas PNA', 'stock', 'b3', 'USIM5', 'BRL', 'BVMF'),
    ('CSNA3', 'CSNA3 - CSN', 'stock', 'b3', 'CSNA3', 'BRL', 'BVMF'),
    ('EMBR3', 'EMBR3 - Embraer', 'stock', 'b3', 'EMBR3', 'BRL', 'BVMF'),
    ('KLBN11', 'KLBN11 - Klabin Unit', 'stock', 'b3', 'KLBN11', 'BRL', 'BVMF'),
    ('CYRE3', 'CYRE3 - Cyrela', 'stock', 'b3', 'CYRE3', 'BRL', 'BVMF'),
    ('MULT3', 'MULT3 - Multiplan', 'stock', 'b3', 'MULT3', 'BRL', 'BVMF'),
    ('EQTL3', 'EQTL3 - Equatorial Energia', 'stock', 'b3', 'EQTL3', 'BRL', 'BVMF'),
    ('ENBR3', 'ENBR3 - EDP Brasil', 'stock', 'b3', 'ENBR3', 'BRL', 'BVMF'),
    ('TIMS3', 'TIMS3 - TIM S.A.', 'stock', 'b3', 'TIMS3', 'BRL', 'BVMF'),
    ('VIVT3', 'VIVT3 - Telefônica Brasil', 'stock', 'b3', 'VIVT3', 'BRL', 'BVMF'),
    ('ENGI11', 'ENGI11 - Energisa Unit', 'stock', 'b3', 'ENGI11', 'BRL', 'BVMF'),
    ('BRFS3', 'BRFS3 - BRF S.A.', 'stock', 'b3', 'BRFS3', 'BRL', 'BVMF'),
    ('NTCO3', 'NTCO3 - Natura &Co', 'stock', 'b3', 'NTCO3', 'BRL', 'BVMF')
ON CONFLICT (ticker, asset_type) DO NOTHING;

-- Seed US stocks (USD, NASDAQ as catch-all label).
INSERT INTO assets (ticker, name, asset_type, source, external_id, currency, market) VALUES
    ('AAPL', 'AAPL - Apple Inc.', 'stock', 'nasdaq', 'AAPL', 'USD', 'NASDAQ'),
    ('MSFT', 'MSFT - Microsoft Corp.', 'stock', 'nasdaq', 'MSFT', 'USD', 'NASDAQ'),
    ('GOOGL', 'GOOGL - Alphabet Class A', 'stock', 'nasdaq', 'GOOGL', 'USD', 'NASDAQ'),
    ('AMZN', 'AMZN - Amazon.com', 'stock', 'nasdaq', 'AMZN', 'USD', 'NASDAQ'),
    ('NVDA', 'NVDA - NVIDIA Corp.', 'stock', 'nasdaq', 'NVDA', 'USD', 'NASDAQ'),
    ('TSLA', 'TSLA - Tesla Inc.', 'stock', 'nasdaq', 'TSLA', 'USD', 'NASDAQ'),
    ('META', 'META - Meta Platforms', 'stock', 'nasdaq', 'META', 'USD', 'NASDAQ'),
    ('JPM', 'JPM - JPMorgan Chase', 'stock', 'nasdaq', 'JPM', 'USD', 'NASDAQ'),
    ('V', 'V - Visa Inc.', 'stock', 'nasdaq', 'V', 'USD', 'NASDAQ'),
    ('MA', 'MA - Mastercard Inc.', 'stock', 'nasdaq', 'MA', 'USD', 'NASDAQ'),
    ('JNJ', 'JNJ - Johnson & Johnson', 'stock', 'nasdaq', 'JNJ', 'USD', 'NASDAQ'),
    ('WMT', 'WMT - Walmart Inc.', 'stock', 'nasdaq', 'WMT', 'USD', 'NASDAQ'),
    ('PG', 'PG - Procter & Gamble', 'stock', 'nasdaq', 'PG', 'USD', 'NASDAQ'),
    ('XOM', 'XOM - Exxon Mobil', 'stock', 'nasdaq', 'XOM', 'USD', 'NASDAQ'),
    ('BAC', 'BAC - Bank of America', 'stock', 'nasdaq', 'BAC', 'USD', 'NASDAQ'),
    ('KO', 'KO - Coca-Cola Company', 'stock', 'nasdaq', 'KO', 'USD', 'NASDAQ'),
    ('PEP', 'PEP - PepsiCo Inc.', 'stock', 'nasdaq', 'PEP', 'USD', 'NASDAQ'),
    ('NFLX', 'NFLX - Netflix Inc.', 'stock', 'nasdaq', 'NFLX', 'USD', 'NASDAQ'),
    ('DIS', 'DIS - Walt Disney Co.', 'stock', 'nasdaq', 'DIS', 'USD', 'NASDAQ'),
    ('AMD', 'AMD - Advanced Micro Devices', 'stock', 'nasdaq', 'AMD', 'USD', 'NASDAQ'),
    ('INTC', 'INTC - Intel Corp.', 'stock', 'nasdaq', 'INTC', 'USD', 'NASDAQ'),
    ('ORCL', 'ORCL - Oracle Corp.', 'stock', 'nasdaq', 'ORCL', 'USD', 'NASDAQ'),
    ('CSCO', 'CSCO - Cisco Systems', 'stock', 'nasdaq', 'CSCO', 'USD', 'NASDAQ'),
    ('CRM', 'CRM - Salesforce Inc.', 'stock', 'nasdaq', 'CRM', 'USD', 'NASDAQ'),
    ('ADBE', 'ADBE - Adobe Inc.', 'stock', 'nasdaq', 'ADBE', 'USD', 'NASDAQ'),
    ('T', 'T - AT&T Inc.', 'stock', 'nasdaq', 'T', 'USD', 'NASDAQ'),
    ('VZ', 'VZ - Verizon Communications', 'stock', 'nasdaq', 'VZ', 'USD', 'NASDAQ'),
    ('NKE', 'NKE - Nike Inc.', 'stock', 'nasdaq', 'NKE', 'USD', 'NASDAQ'),
    ('MCD', 'MCD - McDonalds Corp.', 'stock', 'nasdaq', 'MCD', 'USD', 'NASDAQ'),
    ('ABT', 'ABT - Abbott Laboratories', 'stock', 'nasdaq', 'ABT', 'USD', 'NASDAQ')
ON CONFLICT (ticker, asset_type) DO NOTHING;

-- Seed 20 cryptos (USD, CoinGecko ids).
INSERT INTO assets (ticker, name, asset_type, source, external_id, currency, market) VALUES
    ('BTC', 'BTC - Bitcoin', 'crypto', 'coingecko', 'bitcoin', 'USD', 'CRYPTO'),
    ('ETH', 'ETH - Ethereum', 'crypto', 'coingecko', 'ethereum', 'USD', 'CRYPTO'),
    ('USDT', 'USDT - Tether', 'crypto', 'coingecko', 'tether', 'USD', 'CRYPTO'),
    ('BNB', 'BNB - BNB', 'crypto', 'coingecko', 'binancecoin', 'USD', 'CRYPTO'),
    ('SOL', 'SOL - Solana', 'crypto', 'coingecko', 'solana', 'USD', 'CRYPTO'),
    ('XRP', 'XRP - XRP', 'crypto', 'coingecko', 'ripple', 'USD', 'CRYPTO'),
    ('USDC', 'USDC - USD Coin', 'crypto', 'coingecko', 'usd-coin', 'USD', 'CRYPTO'),
    ('ADA', 'ADA - Cardano', 'crypto', 'coingecko', 'cardano', 'USD', 'CRYPTO'),
    ('DOGE', 'DOGE - Dogecoin', 'crypto', 'coingecko', 'dogecoin', 'USD', 'CRYPTO'),
    ('TRX', 'TRX - TRON', 'crypto', 'coingecko', 'tron', 'USD', 'CRYPTO'),
    ('AVAX', 'AVAX - Avalanche', 'crypto', 'coingecko', 'avalanche-2', 'USD', 'CRYPTO'),
    ('DOT', 'DOT - Polkadot', 'crypto', 'coingecko', 'polkadot', 'USD', 'CRYPTO'),
    ('MATIC', 'MATIC - Polygon', 'crypto', 'coingecko', 'matic-network', 'USD', 'CRYPTO'),
    ('LINK', 'LINK - Chainlink', 'crypto', 'coingecko', 'chainlink', 'USD', 'CRYPTO'),
    ('LTC', 'LTC - Litecoin', 'crypto', 'coingecko', 'litecoin', 'USD', 'CRYPTO'),
    ('BCH', 'BCH - Bitcoin Cash', 'crypto', 'coingecko', 'bitcoin-cash', 'USD', 'CRYPTO'),
    ('NEAR', 'NEAR - NEAR Protocol', 'crypto', 'coingecko', 'near', 'USD', 'CRYPTO'),
    ('UNI', 'UNI - Uniswap', 'crypto', 'coingecko', 'uniswap', 'USD', 'CRYPTO'),
    ('ATOM', 'ATOM - Cosmos', 'crypto', 'coingecko', 'cosmos', 'USD', 'CRYPTO'),
    ('ETC', 'ETC - Ethereum Classic', 'crypto', 'coingecko', 'ethereum-classic', 'USD', 'CRYPTO')
ON CONFLICT (ticker, asset_type) DO NOTHING;
