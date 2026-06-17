"use client";

import { useEffect, useState } from "react";
import { Box, Flex, Text } from "@chakra-ui/react";
import { api } from "@/lib/api";
import type { AssetType, ApiResponse, PriceQuote } from "@/types";

interface Props {
  ticker: string;
  assetType: AssetType;
  quantity: string | null;
  amountInvested: string;
  purchasePrice?: string;
  currency?: string;
}

type State =
  | { kind: "loading" }
  | { kind: "ok"; quote: PriceQuote }
  | { kind: "missing" }
  | { kind: "error" };

function formatMoney(value: number, currency: string) {
  if (!Number.isFinite(value)) return "—";
  try {
    return value.toLocaleString("pt-BR", {
      style: "currency",
      currency: currency || "BRL",
      maximumFractionDigits: 2,
    });
  } catch {
    return `${currency || ""} ${value.toLocaleString("pt-BR", {
      maximumFractionDigits: 2,
    })}`.trim();
  }
}

function formatPercent(value: number) {
  if (!Number.isFinite(value)) return "—";
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toLocaleString("pt-BR", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}%`;
}

// PositionCell substitui o PriceBadge na tabela de investimentos: mostra
// (1) valor atual da posição (quantidade * preço atual) e
// (2) ganho/perda absoluto + percentual contra amount_invested.
// Quando dados faltam (sem quantity, sem purchase_price, sem cotação), degrada
// silenciosamente em "—" — nunca renderiza valores fabricados.
export function PositionCell({
  ticker,
  assetType,
  quantity,
  amountInvested,
  purchasePrice,
  currency,
}: Props) {
  const [state, setState] = useState<State>({ kind: "loading" });

  useEffect(() => {
    let cancelled = false;
    async function go() {
      if (!ticker) {
        setState({ kind: "missing" });
        return;
      }
      setState({ kind: "loading" });
      try {
        const params = new URLSearchParams({ type: assetType });
        const res = await api.get<ApiResponse<PriceQuote>>(
          `/prices/${encodeURIComponent(ticker)}?${params.toString()}`,
        );
        if (cancelled) return;
        if (res && res.data) {
          setState({ kind: "ok", quote: res.data });
        } else {
          setState({ kind: "missing" });
        }
      } catch (err) {
        if (cancelled) return;
        const msg = err instanceof Error ? err.message.toLowerCase() : "";
        if (msg.includes("not found") || msg.includes("404")) {
          setState({ kind: "missing" });
        } else {
          setState({ kind: "error" });
        }
      }
    }
    go();
    return () => {
      cancelled = true;
    };
  }, [ticker, assetType]);

  if (state.kind === "loading") {
    return (
      <Box>
        <Text fontSize="sm" color="gray.500">
          …
        </Text>
      </Box>
    );
  }

  if (state.kind !== "ok") {
    return (
      <Box>
        <Text fontSize="sm" color="gray.400" lineHeight="1.1">
          —
        </Text>
        <Text fontSize="xs" color="gray.500" mt="0.5">
          {state.kind === "missing" ? "preço indisponível" : "erro ao carregar"}
        </Text>
      </Box>
    );
  }

  const quote = state.quote;
  const rowCurrency = currency || quote.currency || "BRL";
  const currentPrice = Number(quote.price);
  const qty = quantity == null ? null : Number(quantity);
  const invested = Number(amountInvested);
  const purchase = purchasePrice ? Number(purchasePrice) : null;

  // Sem quantidade não dá pra calcular valor da posição — mostra unitário.
  if (qty == null || !Number.isFinite(qty)) {
    return (
      <Box>
        <Text
          fontSize="sm"
          color="white"
          fontWeight="semibold"
          lineHeight="1.1"
        >
          {formatMoney(currentPrice, rowCurrency)}
        </Text>
        <Text fontSize="xs" color="gray.500" mt="0.5">
          —
        </Text>
      </Box>
    );
  }

  const currentValue = qty * currentPrice;

  // Sem purchase_price (dado legado), só mostra valor da posição sem ganho/perda.
  if (purchase == null || !Number.isFinite(purchase) || !Number.isFinite(invested)) {
    return (
      <Box>
        <Text
          fontSize="sm"
          color="white"
          fontWeight="semibold"
          lineHeight="1.1"
        >
          {formatMoney(currentValue, rowCurrency)}
        </Text>
        <Text fontSize="xs" color="gray.500" mt="0.5">
          —
        </Text>
      </Box>
    );
  }

  const pnl = currentValue - invested;
  const pnlPct = invested > 0 ? (pnl / invested) * 100 : 0;
  // Tolerância pra evitar piscar verde/vermelho em variação centavo.
  const pnlColor =
    pnl > 0.005 ? "#86efac" : pnl < -0.005 ? "#fca5a5" : "gray.400";

  return (
    <Box>
      <Text fontSize="sm" color="white" fontWeight="semibold" lineHeight="1.1">
        {formatMoney(currentValue, rowCurrency)}
      </Text>
      <Flex align="center" gap="1" mt="0.5">
        <Text fontSize="xs" color={pnlColor} fontWeight="medium">
          {pnl >= 0 ? "+" : ""}
          {formatMoney(pnl, rowCurrency)}
        </Text>
        <Text fontSize="xs" color={pnlColor}>
          ({formatPercent(pnlPct)})
        </Text>
      </Flex>
    </Box>
  );
}
