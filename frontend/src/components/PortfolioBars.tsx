"use client";

import { Box, Flex, Text } from "@chakra-ui/react";
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export interface PortfolioBar {
  name: string;
  total: number;
  type: "real" | "simulated";
}

interface Props {
  data: PortfolioBar[];
}

function formatBRL(value: number) {
  return value.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 0,
  });
}

const TYPE_COLOR: Record<string, string> = {
  real: "#0ea5e9",
  simulated: "#a855f7",
};

export function PortfolioBars({ data }: Props) {
  if (data.length === 0) {
    return (
      <Flex
        h="260px"
        align="center"
        justify="center"
        direction="column"
        gap="2"
        color="gray.500"
      >
        <Box w="160px" h="4px" bg="gray.700" borderRadius="full" />
        <Box w="120px" h="4px" bg="gray.700" borderRadius="full" />
        <Box w="80px" h="4px" bg="gray.700" borderRadius="full" />
        <Text fontSize="sm" mt="2">
          Crie um portfolio para ver a comparação
        </Text>
      </Flex>
    );
  }

  // Recharts horizontal bar: layout="vertical" + xAxis number + yAxis category
  return (
    <Box h="260px" w="100%" minW="0">
      <ResponsiveContainer width="100%" height={260} debounce={50}>
        <BarChart
          layout="vertical"
          data={data}
          margin={{ top: 10, right: 24, bottom: 0, left: 0 }}
        >
          <XAxis
            type="number"
            stroke="#6b7280"
            tick={{ fontSize: 11, fill: "#9ca3af" }}
            tickFormatter={(v: number) =>
              v >= 1000 ? `${(v / 1000).toFixed(0)}k` : `${v}`
            }
            axisLine={false}
            tickLine={false}
          />
          <YAxis
            type="category"
            dataKey="name"
            stroke="#6b7280"
            tick={{ fontSize: 12, fill: "#cbd5e1" }}
            axisLine={false}
            tickLine={false}
            width={110}
          />
          <Tooltip
            cursor={{ fill: "rgba(75, 85, 99, 0.2)" }}
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
          <Bar dataKey="total" radius={[0, 6, 6, 0]} isAnimationActive={false}>
            {data.map((d, idx) => (
              <Cell key={idx} fill={TYPE_COLOR[d.type] ?? "#0ea5e9"} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </Box>
  );
}
