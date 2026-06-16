"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Box,
  Button,
  Heading,
  SimpleGrid,
  Text,
  Badge,
  Flex,
  Spinner,
  Center,
  HStack,
  Stack,
} from "@chakra-ui/react";
import NextLink from "next/link";
import { api } from "@/lib/api";
import { toaster } from "@/lib/toaster";
import type {
  Portfolio,
  PortfolioWithInvestments,
  ApiResponse,
} from "@/types";
import { ASSET_COLORS, ASSET_LABEL } from "@/components/AllocationDonut";
import {
  EyeIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from "@/components/Icons";

type Filter = "all" | "real" | "simulated";

function formatBRL(value: number) {
  return value.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 2,
  });
}

// A barra horizontal mostrando a alocação por classe de ativo
// dentro do card de cada portfólio (substitui mini-chart por algo
// SVG-livre, legível em 200px de largura).
function AllocationBar({
  data,
}: {
  data: { asset_type: string; value: number }[];
}) {
  const total = data.reduce((s, d) => s + d.value, 0);
  if (total === 0) {
    return (
      <Box h="6px" bg="gray.700" borderRadius="full" overflow="hidden" />
    );
  }
  return (
    <Flex h="6px" borderRadius="full" overflow="hidden" bg="gray.700">
      {data.map((d) => {
        const pct = (d.value / total) * 100;
        return (
          <Box
            key={d.asset_type}
            h="100%"
            w={`${pct}%`}
            bg={ASSET_COLORS[d.asset_type] ?? "#6b7280"}
          />
        );
      })}
    </Flex>
  );
}

export default function PortfoliosPage() {
  const [portfolios, setPortfolios] = useState<PortfolioWithInvestments[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<Filter>("all");
  const router = useRouter();

  const fetchPortfolios = useCallback(async () => {
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
    } catch {
      toaster.create({
        title: "Falha ao carregar portfólios",
        type: "error",
        duration: 3000,
      });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPortfolios();
  }, [fetchPortfolios]);

  const handleDelete = async (id: string) => {
    if (!confirm("Tem certeza que quer deletar esse portfólio?")) return;

    try {
      await api.delete(`/portfolios/${id}`);
      setPortfolios((prev) => prev.filter((p) => p.id !== id));
      toaster.create({
        title: "Portfólio deletado",
        type: "success",
        duration: 2000,
      });
    } catch {
      toaster.create({
        title: "Falha ao deletar portfólio",
        type: "error",
        duration: 3000,
      });
    }
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  const counts = {
    all: portfolios.length,
    real: portfolios.filter((p) => p.type === "real").length,
    simulated: portfolios.filter((p) => p.type === "simulated").length,
  };

  const visible =
    filter === "all" ? portfolios : portfolios.filter((p) => p.type === filter);

  const tabs: { id: Filter; label: string }[] = [
    { id: "all", label: "Todos" },
    { id: "real", label: "Reais" },
    { id: "simulated", label: "Simulados" },
  ];

  return (
    <Stack gap="6">
      <Flex justify="space-between" align="end" wrap="wrap" gap="4">
        <Box>
          <Heading size="xl" color="white">
            Portfólios
          </Heading>
          <Text color="gray.400" mt="1">
            Suas carteiras reais e simuladas
          </Text>
        </Box>
        <NextLink href="/dashboard/portfolios/new">
          <Button colorPalette="blue">
            <PlusIcon size={16} />
            <Text ml="2">Novo portfólio</Text>
          </Button>
        </NextLink>
      </Flex>

      {/* Filter tabs */}
      {portfolios.length > 0 && (
        <HStack
          gap="1"
          bg="gray.800"
          border="1px solid"
          borderColor="gray.700"
          borderRadius="lg"
          p="1"
          w="fit-content"
        >
          {tabs.map((t) => {
            const active = filter === t.id;
            return (
              <Button
                key={t.id}
                size="sm"
                variant={active ? "solid" : "ghost"}
                colorPalette={active ? "blue" : "gray"}
                onClick={() => setFilter(t.id)}
                px="4"
              >
                <Text>{t.label}</Text>
                <Box
                  ml="2"
                  px="2"
                  py="0.5"
                  fontSize="xs"
                  fontWeight="bold"
                  bg={active ? "rgba(255,255,255,0.18)" : "gray.700"}
                  color={active ? "white" : "gray.400"}
                  borderRadius="full"
                >
                  {counts[t.id]}
                </Box>
              </Button>
            );
          })}
        </HStack>
      )}

      {visible.length === 0 ? (
        <Box
          bg="gray.800"
          border="1px solid"
          borderColor="gray.700"
          borderRadius="lg"
          p="10"
          textAlign="center"
        >
          <Heading size="md" color="white" mb="2">
            {portfolios.length === 0
              ? "Nenhum portfólio ainda"
              : `Nenhum portfólio ${filter === "real" ? "real" : "simulado"}`}
          </Heading>
          <Text color="gray.400" mb="5">
            {portfolios.length === 0
              ? "Crie sua primeira carteira para começar a acompanhar."
              : "Troque o filtro ou crie um novo."}
          </Text>
          <NextLink href="/dashboard/portfolios/new">
            <Button colorPalette="blue">
              <PlusIcon size={16} />
              <Text ml="2">Criar portfólio</Text>
            </Button>
          </NextLink>
        </Box>
      ) : (
        <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap="4">
          {visible.map((portfolio) => {
            const totalInvested = portfolio.investments.reduce(
              (s, i) => s + (Number(i.amount_invested) || 0),
              0,
            );
            const allocByType = portfolio.investments.reduce<
              Record<string, number>
            >((acc, i) => {
              acc[i.asset_type] =
                (acc[i.asset_type] ?? 0) + (Number(i.amount_invested) || 0);
              return acc;
            }, {});
            const allocList = Object.entries(allocByType)
              .map(([asset_type, value]) => ({ asset_type, value }))
              .sort((a, b) => b.value - a.value);

            return (
              <Box
                key={portfolio.id}
                className="lift"
                bg="gray.800"
                borderRadius="lg"
                border="1px solid"
                borderColor="gray.700"
                overflow="hidden"
              >
                <Box className={`accent-bar ${portfolio.type}`} />
                <Box p="5">
                  <Flex justify="space-between" align="start" mb="3">
                    <Heading size="sm" color="white" lineClamp={1}>
                      {portfolio.name}
                    </Heading>
                    <Badge
                      colorPalette={portfolio.type === "real" ? "blue" : "purple"}
                      variant={portfolio.type === "real" ? "solid" : "outline"}
                      size="sm"
                    >
                      {portfolio.type}
                    </Badge>
                  </Flex>

                  <Text fontSize="xs" color="gray.500" mb="1">
                    Valor investido
                  </Text>
                  <Heading size="md" color="white" mb="4">
                    {formatBRL(totalInvested)}
                  </Heading>

                  {/* Allocation bar + legend */}
                  {allocList.length > 0 && (
                    <Box mb="4">
                      <AllocationBar data={allocList} />
                      <Flex gap="3" mt="2" wrap="wrap">
                        {allocList.slice(0, 4).map((s) => (
                          <Flex key={s.asset_type} align="center" gap="1.5">
                            <Box
                              w="6px"
                              h="6px"
                              borderRadius="full"
                              bg={ASSET_COLORS[s.asset_type] ?? "#6b7280"}
                            />
                            <Text fontSize="xs" color="gray.400">
                              {ASSET_LABEL[s.asset_type] ?? s.asset_type}
                            </Text>
                          </Flex>
                        ))}
                      </Flex>
                    </Box>
                  )}

                  <HStack
                    gap="3"
                    fontSize="xs"
                    color="gray.400"
                    borderTop="1px solid"
                    borderColor="gray.700"
                    pt="3"
                    mb="4"
                  >
                    <Text>
                      <Text as="span" color="white" fontWeight="bold">
                        {portfolio.investments.length}
                      </Text>{" "}
                      posições
                    </Text>
                    <Text color="gray.600">·</Text>
                    <Text>
                      Criado em{" "}
                      {new Date(portfolio.created_at).toLocaleDateString("pt-BR")}
                    </Text>
                  </HStack>

                  <HStack gap="2">
                    <Button
                      size="xs"
                      colorPalette="blue"
                      flex="1"
                      onClick={() =>
                        router.push(`/dashboard/portfolios/${portfolio.id}`)
                      }
                    >
                      <EyeIcon size={14} />
                      <Text ml="1">Ver</Text>
                    </Button>
                    <Button
                      size="xs"
                      variant="outline"
                      onClick={() =>
                        router.push(`/dashboard/portfolios/${portfolio.id}/edit`)
                      }
                      aria-label="Editar"
                    >
                      <PencilIcon size={14} />
                    </Button>
                    <Button
                      size="xs"
                      variant="outline"
                      colorPalette="red"
                      onClick={() => handleDelete(portfolio.id)}
                      aria-label="Deletar"
                    >
                      <TrashIcon size={14} />
                    </Button>
                  </HStack>
                </Box>
              </Box>
            );
          })}
        </SimpleGrid>
      )}
    </Stack>
  );
}
