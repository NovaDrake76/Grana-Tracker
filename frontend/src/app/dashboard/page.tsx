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
  LayersIcon,
  PlusIcon,
  PortfolioIcon,
  SparkleIcon,
  TrendingUpIcon,
  WalletIcon,
} from "@/components/Icons";
import { useAuth } from "@/context/AuthContext";

function formatBRL(value: number) {
  if (!Number.isFinite(value)) return "—";
  return value.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 2,
  });
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

  if (loading) {
    return (
      <Center h="60vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  const totalPortfolios = portfolios.length;
  const realCount = portfolios.filter((p) => p.type === "real").length;
  const simulatedCount = totalPortfolios - realCount;
  const allInvestments = portfolios.flatMap((p) => p.investments);
  const totalHoldings = allInvestments.length;
  const totalInvested = allInvestments.reduce(
    (sum, inv) => sum + (Number(inv.amount_invested) || 0),
    0,
  );

  const assetTypeCounts = allInvestments.reduce<Record<string, number>>(
    (acc, inv) => {
      acc[inv.asset_type] = (acc[inv.asset_type] ?? 0) + 1;
      return acc;
    },
    {},
  );

  const recent = [...portfolios]
    .sort((a, b) => b.created_at.localeCompare(a.created_at))
    .slice(0, 3);

  const firstName = user?.name?.split(" ")[0] ?? "investidor";

  return (
    <Stack gap="6">
      <Box className="hero-card" p={{ base: "6", md: "8" }}>
        <Flex
          justify="space-between"
          align={{ base: "start", md: "end" }}
          wrap="wrap"
          gap="4"
          position="relative"
          zIndex="1"
        >
          <Box>
            <Text
              fontSize="sm"
              color="brand.300"
              fontWeight="medium"
              mb="2"
              letterSpacing="0.05em"
            >
              VISÃO GERAL
            </Text>
            <Heading
              size="2xl"
              lineHeight="1.1"
              mb="2"
              className="gradient-text"
            >
              Olá, {firstName} 👋
            </Heading>
            <Text color="gray.400" maxW="md">
              Resumo das suas carteiras reais e simuladas em um único painel.
            </Text>
          </Box>
          <NextLink href="/dashboard/portfolios/new">
            <Button
              colorPalette="blue"
              size="md"
              style={{
                background: "linear-gradient(135deg, #0ea5e9, #0284c7)",
                boxShadow: "0 8px 24px -8px rgba(14, 165, 233, 0.6)",
              }}
            >
              <PlusIcon size={16} />
              <Text ml="2">Novo portfólio</Text>
            </Button>
          </NextLink>
        </Flex>
      </Box>

      <SimpleGrid columns={{ base: 1, sm: 2, lg: 4 }} gap="4">
        <StatCard
          label="Portfólios"
          value={totalPortfolios}
          helper={`${realCount} reais · ${simulatedCount} simulados`}
          icon={<PortfolioIcon size={16} />}
          accent="brand"
        />
        <StatCard
          label="Total investido"
          value={formatBRL(totalInvested)}
          helper="Soma de todos os investimentos"
          icon={<WalletIcon size={16} />}
          accent="gain"
        />
        <StatCard
          label="Posições"
          value={totalHoldings}
          helper={
            totalHoldings === 0
              ? "Nenhuma posição ainda"
              : `Em ${totalPortfolios} portfólio${totalPortfolios === 1 ? "" : "s"}`
          }
          icon={<LayersIcon size={16} />}
          accent="purple"
        />
        <StatCard
          label="Classes de ativo"
          value={Object.keys(assetTypeCounts).length || 0}
          helper={
            Object.entries(assetTypeCounts)
              .map(([k, v]) => `${k} ${v}`)
              .join(" · ") || "—"
          }
          icon={<TrendingUpIcon size={16} />}
          accent="gray"
        />
      </SimpleGrid>

      <Box>
        <Flex justify="space-between" align="center" mb="4">
          <Heading size="md" color="white">
            Portfólios recentes
          </Heading>
          <NextLink href="/dashboard/portfolios">
            <Button size="sm" variant="ghost" colorPalette="blue">
              Ver todos →
            </Button>
          </NextLink>
        </Flex>

        {recent.length === 0 ? (
          <Box
            className="glass-card"
            borderRadius="xl"
            p="12"
            textAlign="center"
          >
            <Flex
              w="64px"
              h="64px"
              mx="auto"
              mb="5"
              align="center"
              justify="center"
              color="brand.300"
              borderRadius="full"
              style={{
                background: "linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(168, 85, 247, 0.15))",
                boxShadow: "0 0 40px -8px rgba(14, 165, 233, 0.4)",
              }}
            >
              <SparkleIcon size={32} />
            </Flex>
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
                    className="glass-card lift"
                    borderRadius="xl"
                    overflow="hidden"
                  >
                    <Box
                      className={`accent-bar ${p.type}`}
                    />
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
                      <HStack
                        gap="4"
                        fontSize="xs"
                        color="gray.400"
                        mb="3"
                      >
                        <Text>
                          <Text as="span" color="white" fontWeight="bold">
                            {p.investments.length}
                          </Text>{" "}
                          posições
                        </Text>
                        <Text>
                          <Text as="span" color="gain" fontWeight="bold">
                            {formatBRL(invested)}
                          </Text>
                        </Text>
                      </HStack>
                      <Text fontSize="xs" color="gray.500">
                        Criado em{" "}
                        {new Date(p.created_at).toLocaleDateString("pt-BR")}
                      </Text>
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
