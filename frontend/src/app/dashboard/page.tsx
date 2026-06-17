"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Box,
  Button,
  Center,
  Flex,
  Heading,
  HStack,
  SimpleGrid,
  Spinner,
  Stack,
  Text,
  Badge,
} from "@chakra-ui/react";
import NextLink from "next/link";
import { api } from "@/lib/api";
import type {
  ApiResponse,
  Portfolio,
  PortfolioWithInvestments,
} from "@/types";
import { StatCard } from "@/components/StatCard";
import {
  AllocationDonut,
  AllocationLegend,
  type AllocationSlice,
} from "@/components/AllocationDonut";
import { PortfolioBars, type PortfolioBar } from "@/components/PortfolioBars";
import {
  LayersIcon,
  PlusIcon,
  PortfolioIcon,
  TrendingUpIcon,
  WalletIcon,
} from "@/components/Icons";
import { useAuth } from "@/context/AuthContext";
import { usePriceMap, priceKey } from "@/hooks/usePriceMap";

function formatBRL(value: number) {
  if (!Number.isFinite(value)) return "—";
  return value.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 2,
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

export default function DashboardPage() {
  const { user } = useAuth();
  const [loading, setLoading] = useState(true);
  const [portfolios, setPortfolios] = useState<PortfolioWithInvestments[]>([]);

  const load = useCallback(async () => {
    try {
      const listRes = await api.get<ApiResponse<Portfolio[]>>("/portfolios");
      const list = listRes.data;
      if (list.length === 0) {
        setPortfolios([]);
        return;
      }
      const details = await Promise.all(
        list.map((p) =>
          api
            .get<ApiResponse<PortfolioWithInvestments>>(`/portfolios/${p.id}`)
            .then((r) => r.data)
            .catch(() => ({ ...p, investments: [] }) as PortfolioWithInvestments),
        ),
      );
      setPortfolios(details);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // IMPORTANTE: hooks (useState, useEffect, useCallback, usePriceMap, ...) precisam
  // ser chamados na MESMA ORDEM em todo render — Rules of Hooks. Por isso o
  // usePriceMap fica AQUI, antes do early return de loading. Se ele estivesse
  // depois do `if (loading) return`, o primeiro render (loading=true) chamaria
  // 5 hooks e o segundo (loading=false) chamaria 6, e o React estoura com
  // "Rendered more hooks than during the previous render".
  const allInvestments = portfolios.flatMap((p) => p.investments);
  const {
    map: priceMap,
    loading: pricesLoading,
    error: pricesError,
  } = usePriceMap(allInvestments);

  if (loading) {
    return (
      <Center h="60vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  // Agregados
  const totalPortfolios = portfolios.length;
  const realCount = portfolios.filter((p) => p.type === "real").length;
  const simulatedCount = totalPortfolios - realCount;
  const totalHoldings = allInvestments.length;
  const totalInvested = allInvestments.reduce(
    (sum, inv) => sum + (Number(inv.amount_invested) || 0),
    0,
  );

  // Agrupa patrimônio atual e investido por moeda (BRL/USD). Só posições com
  // quantity definida contribuem pro patrimônio — sem quantidade não dá pra
  // computar valor de mercado de forma honesta.
  const byCurrency: Record<string, { atual: number; investido: number }> = {};
  for (const inv of allInvestments) {
    const cur = (inv.currency || "BRL").toUpperCase();
    if (!byCurrency[cur]) byCurrency[cur] = { atual: 0, investido: 0 };
    byCurrency[cur].investido += Number(inv.amount_invested) || 0;
    const qty = inv.quantity == null ? null : Number(inv.quantity);
    const quote = priceMap[priceKey(inv.ticker, inv.asset_type)];
    if (qty != null && Number.isFinite(qty) && quote) {
      byCurrency[cur].atual += qty * quote.price;
    }
  }
  const brl = byCurrency["BRL"] ?? { atual: 0, investido: 0 };
  const usd = byCurrency["USD"] ?? { atual: 0, investido: 0 };
  const hasUsd = usd.investido > 0 || usd.atual > 0;
  const totalAtualBRL = brl.atual;
  const totalGanhoBRL = brl.atual - brl.investido;
  const totalGanhoPct =
    brl.investido > 0 ? (totalGanhoBRL / brl.investido) * 100 : 0;
  const hasAnyPrice = Object.keys(priceMap).length > 0;
  const showPatrimonio = hasAnyPrice && !pricesError;
  const ganhoColor =
    totalGanhoBRL > 0.005
      ? "gain"
      : totalGanhoBRL < -0.005
        ? "loss"
        : "gray";

  // Alocação por classe de ativo (em valor investido)
  const allocByType = allInvestments.reduce<Record<string, number>>(
    (acc, inv) => {
      acc[inv.asset_type] =
        (acc[inv.asset_type] ?? 0) + (Number(inv.amount_invested) || 0);
      return acc;
    },
    {},
  );
  const allocSlices: AllocationSlice[] = Object.entries(allocByType)
    .map(([asset_type, value]) => ({ asset_type, value }))
    .sort((a, b) => b.value - a.value);

  // Barras: total investido por portfolio
  const portfolioBars: PortfolioBar[] = portfolios
    .map((p) => ({
      name: p.name,
      total: p.investments.reduce(
        (s, i) => s + (Number(i.amount_invested) || 0),
        0,
      ),
      type: p.type,
    }))
    .sort((a, b) => b.total - a.total);

  const avgPerHolding =
    totalHoldings === 0 ? 0 : totalInvested / totalHoldings;

  const recent = [...portfolios]
    .sort((a, b) => b.created_at.localeCompare(a.created_at))
    .slice(0, 3);

  const firstName = user?.name?.split(" ")[0] ?? "investidor";
  const greeting = (() => {
    const h = new Date().getHours();
    if (h < 12) return "Bom dia";
    if (h < 18) return "Boa tarde";
    return "Boa noite";
  })();

  return (
    <Stack gap="6">
      {/* Hero gradiente: greeting + total destacado + CTA */}
      <Box
        position="relative"
        overflow="hidden"
        borderRadius="xl"
        border="1px solid"
        borderColor="gray.700"
        bg="gray.800"
        style={{ background: "linear-gradient(135deg, rgba(14,165,233,0.22) 0%, rgba(168,85,247,0.14) 60%, #1f2937 100%)" }}
        p={{ base: "6", md: "8" }}
      >
        <Box
          position="absolute"
          top="-40px"
          right="-40px"
          w="240px"
          h="240px"
          borderRadius="full"
          style={{ background: "radial-gradient(circle, rgba(14,165,233,0.35) 0%, transparent 70%)" }}
          pointerEvents="none"
        />
        <Flex
          direction={{ base: "column", md: "row" }}
          align={{ base: "start", md: "end" }}
          justify="space-between"
          gap="6"
          position="relative"
        >
          <Box>
            <Text fontSize="sm" color="brand.300" fontWeight="medium" mb="1">
              {greeting}, {firstName} 👋
            </Text>
            <Heading size="xl" color="white" lineHeight="1.1" mb="3">
              {formatBRL(totalInvested)}
            </Heading>
            <Text color="gray.400" fontSize="sm" maxW="md">
              Capital total investido em {totalPortfolios} portfolio
              {totalPortfolios === 1 ? "" : "s"} ({realCount} real
              {realCount === 1 ? "" : "is"}
              {simulatedCount > 0
                ? ` · ${simulatedCount} simulado${simulatedCount === 1 ? "" : "s"}`
                : ""}
              ) distribuídos em {totalHoldings} posição
              {totalHoldings === 1 ? "" : "ões"}.
            </Text>
          </Box>
          <NextLink href="/dashboard/portfolios/new">
            <Button colorPalette="blue" size="md">
              <PlusIcon size={16} />
              <Text ml="2">Novo portfólio</Text>
            </Button>
          </NextLink>
        </Flex>
      </Box>

      {/* Stat cards — agora com Patrimônio atual e Ganho/Perda em destaque.
          Cards originais ficam mais enxutos (helpers curtos) pra caber em
          telas médias sem virar mar de números. */}
      <SimpleGrid columns={{ base: 1, sm: 2, lg: 3, xl: 6 }} gap="4">
        <StatCard
          label="Patrimônio atual"
          value={showPatrimonio ? formatBRL(totalAtualBRL) : "—"}
          helper={
            pricesLoading
              ? "atualizando preços…"
              : !showPatrimonio
                ? "preços indisponíveis"
                : hasUsd
                  ? `+ ${formatMoney(usd.atual, "USD")}`
                  : "soma das posições em BRL"
          }
          icon={<TrendingUpIcon size={16} />}
          accent="brand"
        />
        <StatCard
          label="Ganho / Perda"
          value={
            showPatrimonio
              ? `${formatBRL(totalGanhoBRL)} (${
                  totalGanhoPct >= 0 ? "+" : ""
                }${totalGanhoPct.toLocaleString("pt-BR", {
                  minimumFractionDigits: 2,
                  maximumFractionDigits: 2,
                })}%)`
              : "—"
          }
          helper={
            pricesLoading
              ? "atualizando preços…"
              : !showPatrimonio
                ? "sem cotações no cache"
                : hasUsd
                  ? `USD: ${formatMoney(usd.atual - usd.investido, "USD")}`
                  : "vs. valor investido"
          }
          icon={<WalletIcon size={16} />}
          accent={ganhoColor}
        />
        <StatCard
          label="Portfólios"
          value={totalPortfolios}
          helper={`${realCount} reais · ${simulatedCount} simulados`}
          icon={<PortfolioIcon size={16} />}
          accent="purple"
        />
        <StatCard
          label="Posições"
          value={totalHoldings}
          helper={
            totalHoldings === 0
              ? "Nenhuma posição"
              : `${totalPortfolios} portfólio${totalPortfolios === 1 ? "" : "s"}`
          }
          icon={<LayersIcon size={16} />}
          accent="purple"
        />
        <StatCard
          label="Classes de ativo"
          value={Object.keys(allocByType).length || 0}
          helper={allocSlices.map((s) => s.asset_type).join(" · ") || "—"}
          icon={<TrendingUpIcon size={16} />}
          accent="gain"
        />
        <StatCard
          label="Ticket médio"
          value={formatBRL(avgPerHolding)}
          helper="Por posição"
          icon={<WalletIcon size={16} />}
          accent="gray"
        />
      </SimpleGrid>

      {/* Charts row */}
      <SimpleGrid columns={{ base: 1, lg: 2 }} gap="4">
        <Box
          bg="gray.800"
          border="1px solid"
          borderColor="gray.700"
          borderRadius="lg"
          p="6"
        >
          <Flex justify="space-between" align="start" mb="4">
            <Box>
              <Heading size="sm" color="white">
                Alocação por classe
              </Heading>
              <Text fontSize="xs" color="gray.500" mt="1">
                Distribuição do total investido
              </Text>
            </Box>
          </Flex>
          <AllocationDonut data={allocSlices} total={totalInvested} />
          {allocSlices.length > 0 && (
            <Box mt="4" pt="4" borderTop="1px solid" borderColor="gray.700">
              <AllocationLegend data={allocSlices} />
            </Box>
          )}
        </Box>

        <Box
          bg="gray.800"
          border="1px solid"
          borderColor="gray.700"
          borderRadius="lg"
          p="6"
        >
          <Flex justify="space-between" align="start" mb="4">
            <Box>
              <Heading size="sm" color="white">
                Total por portfolio
              </Heading>
              <Text fontSize="xs" color="gray.500" mt="1">
                Comparativo entre carteiras
              </Text>
            </Box>
            <HStack gap="3" fontSize="xs">
              <Flex align="center" gap="1">
                <Box w="8px" h="8px" borderRadius="sm" bg="#0ea5e9" />
                <Text color="gray.400">real</Text>
              </Flex>
              <Flex align="center" gap="1">
                <Box w="8px" h="8px" borderRadius="sm" bg="#a855f7" />
                <Text color="gray.400">simulado</Text>
              </Flex>
            </HStack>
          </Flex>
          <PortfolioBars data={portfolioBars} />
        </Box>
      </SimpleGrid>

      {/* Recent portfolios */}
      <Box>
        <Flex justify="space-between" align="center" mb="4">
          <Heading size="md" color="white">
            Portfólios recentes
          </Heading>
          <NextLink href="/dashboard/portfolios">
            <Button size="sm" variant="ghost" colorPalette="blue">
              Ver todos
            </Button>
          </NextLink>
        </Flex>

        {recent.length === 0 ? (
          <Box
            bg="gray.800"
            border="1px solid"
            borderColor="gray.700"
            borderRadius="md"
            p="10"
            textAlign="center"
          >
            <Heading size="md" color="white" mb="2">
              Vamos começar?
            </Heading>
            <Text color="gray.400" mb="5">
              Crie seu primeiro portfólio para acompanhar investimentos.
            </Text>
            <NextLink href="/dashboard/portfolios/new">
              <Button colorPalette="blue">
                <PlusIcon size={16} />
                <Text ml="2">Criar portfólio</Text>
              </Button>
            </NextLink>
          </Box>
        ) : (
          <SimpleGrid columns={{ base: 1, md: 3 }} gap="4">
            {recent.map((p) => {
              const invested = p.investments.reduce(
                (s, i) => s + (Number(i.amount_invested) || 0),
                0,
              );
              return (
                <NextLink
                  key={p.id}
                  href={`/dashboard/portfolios/${p.id}`}
                  style={{ display: "block" }}
                >
                  <Box
                    className="lift"
                    bg="gray.800"
                    border="1px solid"
                    borderColor="gray.700"
                    borderRadius="lg"
                    overflow="hidden"
                  >
                    <Box className={`accent-bar ${p.type}`} />
                    <Box p="5">
                      <Flex justify="space-between" align="start" mb="3">
                        <Heading size="sm" color="white" lineClamp={1}>
                          {p.name}
                        </Heading>
                        <Badge
                          colorPalette={p.type === "real" ? "blue" : "purple"}
                          variant={p.type === "real" ? "solid" : "outline"}
                          size="sm"
                        >
                          {p.type}
                        </Badge>
                      </Flex>
                      <Text fontSize="xs" color="gray.500" mb="2">
                        Valor investido
                      </Text>
                      <Heading size="md" color="white" mb="3">
                        {formatBRL(invested)}
                      </Heading>
                      <HStack
                        gap="3"
                        fontSize="xs"
                        color="gray.400"
                        borderTop="1px solid"
                        borderColor="gray.700"
                        pt="3"
                      >
                        <Text>
                          <Text as="span" color="white" fontWeight="bold">
                            {p.investments.length}
                          </Text>{" "}
                          posições
                        </Text>
                        <Text color="gray.600">·</Text>
                        <Text>
                          Criado em{" "}
                          {new Date(p.created_at).toLocaleDateString("pt-BR")}
                        </Text>
                      </HStack>
                    </Box>
                  </Box>
                </NextLink>
              );
            })}
          </SimpleGrid>
        )}
      </Box>
    </Stack>
  );
}
