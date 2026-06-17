"use client";

import { useEffect, useState } from "react";
import { Box, Text } from "@chakra-ui/react";
import { api } from "@/lib/api";
import type { AssetType, ApiResponse, PriceQuote } from "@/types";

interface Props {
  ticker: string;
  assetType: AssetType;
  currency?: string;
}

type State =
  | { kind: "loading" }
  | { kind: "ok"; quote: PriceQuote }
  | { kind: "missing" }
  | { kind: "error" };

function formatPrice(value: string, currency: string) {
  const n = Number(value);
  if (!Number.isFinite(n)) return value;
  try {
    return n.toLocaleString("pt-BR", {
      style: "currency",
      currency: currency || "BRL",
      maximumFractionDigits: 2,
    });
  } catch {
    return `${currency || ""} ${n.toLocaleString("pt-BR", {
      maximumFractionDigits: 2,
    })}`.trim();
  }
}

function freshness(updatedAt: string): string {
  const d = new Date(updatedAt);
  if (Number.isNaN(d.getTime())) return "";
  const now = Date.now();
  const diffMs = now - d.getTime();
  const diffMin = Math.round(diffMs / 60000);
  if (diffMin >= 30) {
    try {
      const rtf = new Intl.RelativeTimeFormat("pt-BR", { numeric: "auto" });
      if (diffMin < 60) return `atualizado ${rtf.format(-diffMin, "minute")}`;
      const diffH = Math.round(diffMin / 60);
      if (diffH < 24) return `atualizado ${rtf.format(-diffH, "hour")}`;
      const diffD = Math.round(diffH / 24);
      return `atualizado ${rtf.format(-diffD, "day")}`;
    } catch {
      return `atualizado ${diffMin}min atrás`;
    }
  }
  const hh = d.getHours().toString().padStart(2, "0");
  const mm = d.getMinutes().toString().padStart(2, "0");
  return `atualizado ${hh}:${mm}`;
}

export function PriceBadge({ ticker, assetType, currency }: Props) {
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
        const params = new URLSearchParams({ assetType });
        if (assetType) params.set("type", assetType);
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
        // backend signals missing with 404 -> request() throws "not found"
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

  if (state.kind === "ok") {
    return (
      <Box>
        <Text fontSize="sm" color="white" fontWeight="semibold" lineHeight="1.1">
          {formatPrice(state.quote.price, state.quote.currency || currency || "BRL")}
        </Text>
        <Text fontSize="xs" color="gray.500" mt="0.5">
          {freshness(state.quote.updated_at)}
        </Text>
      </Box>
    );
  }

  if (state.kind === "missing") {
    return (
      <Box>
        <Text fontSize="sm" color="gray.400" lineHeight="1.1">
          —
        </Text>
        <Text fontSize="xs" color="gray.500" mt="0.5">
          preço indisponível
        </Text>
      </Box>
    );
  }

  return (
    <Box>
      <Text fontSize="sm" color="gray.400">
        —
      </Text>
    </Box>
  );
}
