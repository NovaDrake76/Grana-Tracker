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

export interface Investment {
  id: string;
  portfolio_id: string;
  ticker: string;
  asset_type: AssetType;
  amount_invested: string;
  quantity: string | null;
  purchase_date: string;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface PortfolioWithInvestments extends Portfolio {
  investments: Investment[];
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
