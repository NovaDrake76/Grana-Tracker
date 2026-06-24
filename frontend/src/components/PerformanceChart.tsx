"use client";

import { useCallback, useEffect, useState } from "react";
import {
  Box,
  Button,
  Center,
  Flex,
  Heading,
  HStack,
  Spinner,
  Text,
} from "@chakra-ui/react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api } from "@/lib/api";
import type {
  ApiResponse,
  PortfolioHistoryPoint,
  PortfolioHistoryResponse,
} from "@/types";

// US06 — Histórico Patrimonial. Lê snapshots diários gerados pelo cron de
// backend e plota a evolução do patrimônio total do portfolio na janela
// selecionada (7d, 30d, 90d). O backend devolve `value` como string pra
// preservar precisão; o Number() acontece apenas pra eixos/tooltip.

type Period = "7d" | "30d" | "90d";

const PERIODS: { value: Period; label: string }[] = [
  { value: "7d", label: "7 dias" },
  { value: "30d", label: "30 dias" },
  { value: "90d", label: "90 dias" },
];

interface PerformanceChartProps {
  portfolioId: string;
}

interface ChartPoint {
  date: string;
  value: number;
}

function formatDateShort(iso: string) {
  // ISO vem como YYYY-MM-DD ou RFC3339; só precisamos do "DD/MM" pro eixo X.
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" });
}

function formatDateLong(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

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

function formatAxisMoney(value: number, currency: string) {
  if (!Number.isFinite(value)) return "";
  const abs = Math.abs(value);
  const symbol = currency === "USD" ? "US$" : "R$";
  if (abs >= 1_000_000) {
    return `${symbol} ${(value / 1_000_000).toFixed(1)}M`;
  }
  if (abs >= 1_000) {
    return `${symbol} ${(value / 1_000).toFixed(0)}k`;
  }
  return `${symbol} ${value.toFixed(0)}`;
}

export function PerformanceChart({ portfolioId }: PerformanceChartProps) {
  const [period, setPeriod] = useState<Period>("30d");
  const [points, setPoints] = useState<PortfolioHistoryPoint[]>([]);
  const [currency, setCurrency] = useState<string>("BRL");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<ApiResponse<PortfolioHistoryResponse>>(
        `/portfolios/${portfolioId}/history?period=${period}`,
      );
      setPoints(res.data?.points ?? []);
      setCurrency(res.data?.currency || "BRL");
    } catch (err) {
      setError(err instanceof Error ? err.message : "falha ao carregar");
      setPoints([]);
    } finally {
      setLoading(false);
    }
  }, [portfolioId, period]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load();
  }, [load]);

  const chartData: ChartPoint[] = points
    .map((p) => ({ date: p.date, value: Number(p.value) }))
    .filter((p) => Number.isFinite(p.value));

  return (
    <Box
      bg="gray.800"
      border="1px solid"
      borderColor="gray.700"
      borderRadius="lg"
      p="6"
    >
      <Flex
        direction={{ base: "column", md: "row" }}
        justify="space-between"
        align={{ base: "start", md: "center" }}
        gap="3"
        mb="4"
      >
        <Box>
          <Heading size="sm" color="white">
            Histórico patrimonial
          </Heading>
          <Text fontSize="xs" color="gray.500" mt="1">
            Evolução diária do patrimônio total
          </Text>
        </Box>
        <HStack gap="1">
          {PERIODS.map((p) => {
            const active = period === p.value;
            return (
              <Button
                key={p.value}
                size="xs"
                variant={active ? "solid" : "ghost"}
                colorPalette={active ? "blue" : "gray"}
                onClick={() => setPeriod(p.value)}
              >
                {p.label}
              </Button>
            );
          })}
        </HStack>
      </Flex>

      <Box h="280px" w="100%" minW="0">
        {loading ? (
          <Center h="100%">
            <Spinner size="md" color="brand.500" />
          </Center>
        ) : error ? (
          <Center h="100%" flexDirection="column" gap="2" color="gray.500">
            <Text fontSize="sm">Falha ao carregar histórico</Text>
            <Text fontSize="xs">{error}</Text>
          </Center>
        ) : chartData.length === 0 ? (
          <Center h="100%" flexDirection="column" gap="2" color="gray.500" px="4" textAlign="center">
            <Text fontSize="sm">
              Sem dados históricos suficientes — os snapshots são gerados diariamente, volte amanhã
            </Text>
          </Center>
        ) : (
          <ResponsiveContainer width="100%" height={280} debounce={50}>
            <LineChart
              data={chartData}
              margin={{ top: 10, right: 16, bottom: 0, left: 8 }}
            >
              <CartesianGrid stroke="#374151" strokeDasharray="3 3" vertical={false} />
              <XAxis
                dataKey="date"
                stroke="#6b7280"
                tick={{ fontSize: 11, fill: "#9ca3af" }}
                tickFormatter={(v: string) => formatDateShort(v)}
                axisLine={false}
                tickLine={false}
                minTickGap={24}
              />
              <YAxis
                stroke="#6b7280"
                tick={{ fontSize: 11, fill: "#9ca3af" }}
                tickFormatter={(v: number) => formatAxisMoney(v, currency)}
                axisLine={false}
                tickLine={false}
                width={70}
              />
              <Tooltip
                cursor={{ stroke: "#4b5563", strokeWidth: 1 }}
                contentStyle={{
                  background: "#1f2937",
                  border: "1px solid #374151",
                  borderRadius: "6px",
                  fontSize: "12px",
                }}
                labelStyle={{ color: "#fff" }}
                itemStyle={{ color: "#fff" }}
                labelFormatter={(label) => formatDateLong(String(label))}
                formatter={(value) => [
                  formatMoney(Number(value), currency),
                  "Patrimônio",
                ]}
              />
              <Line
                type="monotone"
                dataKey="value"
                stroke="#0ea5e9"
                strokeWidth={2}
                dot={{ r: 3, fill: "#0ea5e9", stroke: "#0ea5e9" }}
                activeDot={{ r: 5 }}
                isAnimationActive={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </Box>
    </Box>
  );
}
