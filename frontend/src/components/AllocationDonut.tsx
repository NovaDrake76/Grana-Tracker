"use client";

import { Box, Flex, Text } from "@chakra-ui/react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

// Paleta dedicada por classe de ativo — espelhada no design system:
// stock=ciano (brand), crypto=violeta, etf=verde gain, index=amber.
// Tons escolhidos pra contrastar bem sobre gray.800 e ficar legível no vídeo.
const ASSET_COLORS: Record<string, string> = {
  stock: "#0ea5e9",
  crypto: "#a855f7",
  etf: "#22c55e",
  index: "#f59e0b",
};

const ASSET_LABEL: Record<string, string> = {
  stock: "Ações",
  crypto: "Cripto",
  etf: "ETFs",
  index: "Índices",
};

export interface AllocationSlice {
  asset_type: string;
  value: number;
}

interface Props {
  data: AllocationSlice[];
  total: number;
  emptyLabel?: string;
}

function formatBRL(value: number) {
  return value.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 0,
  });
}

export function AllocationDonut({ data, total, emptyLabel = "Sem investimentos ainda" }: Props) {
  const hasData = data.length > 0 && total > 0;

  if (!hasData) {
    return (
      <Flex
        h="260px"
        align="center"
        justify="center"
        direction="column"
        gap="2"
        color="gray.500"
      >
        <Box
          w="120px"
          h="120px"
          borderRadius="full"
          border="2px dashed"
          borderColor="gray.700"
        />
        <Text fontSize="sm">{emptyLabel}</Text>
      </Flex>
    );
  }

  const chartData = data.map((s) => ({
    name: ASSET_LABEL[s.asset_type] ?? s.asset_type,
    value: s.value,
    color: ASSET_COLORS[s.asset_type] ?? "#6b7280",
  }));

  return (
    <Box position="relative" h="260px" w="100%" minW="0">
      <ResponsiveContainer width="100%" height={260} debounce={50}>
        <PieChart>
          <Pie
            data={chartData}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="50%"
            innerRadius={70}
            outerRadius={100}
            paddingAngle={2}
            stroke="#111827"
            strokeWidth={3}
            isAnimationActive={false}
          >
            {chartData.map((entry, idx) => (
              <Cell key={idx} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip
            cursor={false}
            contentStyle={{
              background: "#1f2937",
              border: "1px solid #374151",
              borderRadius: "6px",
              fontSize: "12px",
            }}
            labelStyle={{ color: "#fff" }}
            itemStyle={{ color: "#fff" }}
            formatter={(value) => formatBRL(Number(value))}
          />
        </PieChart>
      </ResponsiveContainer>

      {/* Centro do donut: total + label */}
      <Flex
        position="absolute"
        inset="0"
        align="center"
        justify="center"
        direction="column"
        pointerEvents="none"
      >
        <Text fontSize="xs" color="gray.400" textTransform="uppercase" letterSpacing="0.05em">
          Total
        </Text>
        <Text fontSize="xl" fontWeight="bold" color="white" lineHeight="1.1">
          {formatBRL(total)}
        </Text>
      </Flex>
    </Box>
  );
}

export function AllocationLegend({ data }: { data: AllocationSlice[] }) {
  if (data.length === 0) return null;
  const total = data.reduce((s, d) => s + d.value, 0);

  return (
    <Flex direction="column" gap="2">
      {data.map((s) => {
        const pct = total === 0 ? 0 : (s.value / total) * 100;
        return (
          <Flex key={s.asset_type} align="center" justify="space-between" gap="3">
            <Flex align="center" gap="2" minW="0">
              <Box
                w="10px"
                h="10px"
                borderRadius="sm"
                bg={ASSET_COLORS[s.asset_type] ?? "#6b7280"}
              />
              <Text fontSize="sm" color="gray.300" truncate>
                {ASSET_LABEL[s.asset_type] ?? s.asset_type}
              </Text>
            </Flex>
            <Flex align="center" gap="3" flexShrink="0">
              <Text fontSize="sm" color="white" fontWeight="medium">
                {formatBRL(s.value)}
              </Text>
              <Text fontSize="xs" color="gray.500" minW="38px" textAlign="right">
                {pct.toFixed(1)}%
              </Text>
            </Flex>
          </Flex>
        );
      })}
    </Flex>
  );
}

export { ASSET_COLORS, ASSET_LABEL };
