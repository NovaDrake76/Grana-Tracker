export interface User {
  id: string;
  name: string;
  email: string;
  preferred_currency: string;
  created_at: string;
  updated_at: string;
}

export interface Portfolio {
  id: string;
  user_id: string;
  name: string;
  type: "real" | "simulated";
  description: string | null;
  created_at: string;
  updated_at: string;
}

export type AssetType = "stock" | "crypto" | "etf" | "index";

export interface Asset {
  id: string;
  ticker: string;
  name: string;
  asset_type: AssetType;
  currency: string;
  market: string;
  source: string;
}

export interface PriceQuote {
  ticker: string;
  asset_type: AssetType;
  price: string;
  currency: string;
  updated_at: string;
}

export interface Investment {
  id: string;
  portfolio_id: string;
  ticker: string;
  asset_type: AssetType;
  amount_invested: string;
  quantity: string | null;
  purchase_date: string;
  notes: string | null;
  purchase_price?: string;
  currency?: "BRL" | "USD";
  created_at: string;
  updated_at: string;
}

// Resposta de GET /api/prices/historical?ticker=&type=&date= — usada para
// pré-preencher o preço unitário de compra a partir do histórico.
export interface HistoricalPriceResponse {
  ticker: string;
  asset_type: AssetType;
  date: string;
  price: string;
  currency: string;
  source?: string;
}

export interface PortfolioWithInvestments extends Portfolio {
  investments: Investment[];
}

// US06 — pontos da série temporal devolvida por GET /api/portfolios/{id}/history.
// O backend devolve `value` como string pra preservar precisão decimal; o
// componente coage com Number() apenas na hora de renderizar (tooltip/eixo).
export interface PortfolioHistoryPoint {
  date: string;
  value: string;
}

export interface PortfolioHistoryResponse {
  portfolio_id: string;
  currency: string;
  points: PortfolioHistoryPoint[];
}

// US09 — resposta de GET /api/currency/rate?from=&to=.
// `rate` é string (precisão decimal); `fetched_at` é RFC3339.
export interface CurrencyRateResponse {
  from: string;
  to: string;
  rate: string;
  fetched_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface ApiResponse<T> {
  data: T;
  message?: string;
}

export interface ApiError {
  error: string;
  code: string;
}
