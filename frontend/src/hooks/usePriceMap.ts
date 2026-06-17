"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { ApiResponse, AssetType, Investment, PriceQuote } from "@/types";

export interface PriceMapEntry {
  price: number;
  currency: string;
}

export type PriceMap = Record<string, PriceMapEntry>;

// chave estável para o cache: ticker em maiúsculas + tipo de ativo.
export function priceKey(ticker: string, assetType: AssetType): string {
  return `${ticker.toUpperCase()}_${assetType}`;
}

// usePriceMap consulta GET /api/prices/{ticker}?type= para cada par único
// (ticker, asset_type) presente nos investimentos e devolve um mapa em memória.
// O cache vive enquanto o componente raiz estiver montado (sessão da página);
// falhas individuais são engolidas pra que uma cotação ausente não derrube o
// dashboard inteiro — basta render "—" no card correspondente.
export function usePriceMap(investments: Investment[]) {
  const [map, setMap] = useState<PriceMap>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  // serializa a lista de pares pra estabilizar a dependência do efeito.
  const uniqueKey = Array.from(
    new Set(investments.map((i) => priceKey(i.ticker, i.asset_type))),
  )
    .sort()
    .join("|");

  useEffect(() => {
    let cancelled = false;
    if (investments.length === 0) {
      setMap({});
      setLoading(false);
      setError(false);
      return;
    }

    const pairs = Array.from(
      new Map(
        investments.map((i) => [
          priceKey(i.ticker, i.asset_type),
          { ticker: i.ticker, assetType: i.asset_type },
        ]),
      ).values(),
    );

    setLoading(true);
    setError(false);

    (async () => {
      const results = await Promise.all(
        pairs.map(async ({ ticker, assetType }) => {
          try {
            const params = new URLSearchParams({ type: assetType });
            const res = await api.get<ApiResponse<PriceQuote>>(
              `/prices/${encodeURIComponent(ticker)}?${params.toString()}`,
            );
            if (!res?.data) return null;
            const n = Number(res.data.price);
            if (!Number.isFinite(n)) return null;
            return {
              key: priceKey(ticker, assetType),
              entry: { price: n, currency: res.data.currency || "BRL" },
            };
          } catch {
            return null;
          }
        }),
      );
      if (cancelled) return;
      const next: PriceMap = {};
      let anyOk = false;
      for (const r of results) {
        if (r) {
          next[r.key] = r.entry;
          anyOk = true;
        }
      }
      setMap(next);
      setError(!anyOk);
      setLoading(false);
    })();

    return () => {
      cancelled = true;
    };
    // uniqueKey resume o conteúdo do array de forma estável.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [uniqueKey]);

  return { map, loading, error };
}
