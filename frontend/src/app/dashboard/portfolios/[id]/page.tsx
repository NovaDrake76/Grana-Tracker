"use client";

import { useState, useEffect, useCallback, use } from "react";
import { useRouter } from "next/navigation";
import NextLink from "next/link";
import {
  Box,
  Button,
  Badge,
  Center,
  Flex,
  Heading,
  HStack,
  Input,
  NativeSelectField,
  NativeSelectRoot,
  Spinner,
  Stack,
  Table,
  Text,
  Textarea,
  FieldLabel,
  FieldRoot,
  SimpleGrid,
} from "@chakra-ui/react";
import { api } from "@/lib/api";
import { toaster } from "@/lib/toaster";
import type {
  AssetType,
  Investment,
  PortfolioWithInvestments,
  ApiResponse,
} from "@/types";
import {
  AllocationDonut,
  AllocationLegend,
  ASSET_COLORS,
  ASSET_LABEL,
  type AllocationSlice,
} from "@/components/AllocationDonut";
import {
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from "@/components/Icons";
import { TickerAutocomplete } from "@/components/TickerAutocomplete";
import { PriceBadge } from "@/components/PriceBadge";

const ASSET_TYPES: AssetType[] = ["stock", "crypto", "etf", "index"];

function formatBRL(value: number) {
  if (!Number.isFinite(value)) return "—";
  return value.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    maximumFractionDigits: 2,
  });
}

function formatAmount(value: string) {
  if (!value) return "—";
  const n = Number(value);
  if (Number.isNaN(n)) return value;
  return n.toLocaleString("pt-BR", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

function formatQuantity(value: string | null) {
  if (!value) return "—";
  const n = Number(value);
  if (Number.isNaN(n)) return value;
  return n.toLocaleString(undefined, { maximumFractionDigits: 8 });
}

export default function PortfolioDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();

  const [portfolio, setPortfolio] = useState<PortfolioWithInvestments | null>(
    null,
  );
  const [loading, setLoading] = useState(true);

  const [ticker, setTicker] = useState("");
  const [assetType, setAssetType] = useState<AssetType>("stock");
  const [assetTypeTouched, setAssetTypeTouched] = useState(false);
  const [amountInvested, setAmountInvested] = useState("");
  const [quantity, setQuantity] = useState("");
  const [purchaseDate, setPurchaseDate] = useState(
    new Date().toISOString().slice(0, 10),
  );
  const [notes, setNotes] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await api.get<ApiResponse<PortfolioWithInvestments>>(
        `/portfolios/${id}`,
      );
      setPortfolio(res.data);
    } catch {
      toaster.create({
        title: "Portfólio não encontrado",
        type: "error",
        duration: 3000,
      });
      router.push("/dashboard/portfolios");
    } finally {
      setLoading(false);
    }
  }, [id, router]);

  useEffect(() => {
    load();
  }, [load]);

  const resetForm = () => {
    setTicker("");
    setAssetType("stock");
    setAssetTypeTouched(false);
    setAmountInvested("");
    setQuantity("");
    setPurchaseDate(new Date().toISOString().slice(0, 10));
    setNotes("");
  };

  const handleAddInvestment = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      const res = await api.post<ApiResponse<Investment>>(
        `/portfolios/${id}/investments`,
        {
          ticker: ticker.trim().toUpperCase(),
          asset_type: assetType,
          amount_invested: amountInvested,
          quantity: quantity || null,
          purchase_date: purchaseDate,
          notes: notes || null,
        },
      );
      setPortfolio((prev) =>
        prev
          ? { ...prev, investments: [res.data, ...prev.investments] }
          : prev,
      );
      resetForm();
      toaster.create({
        title: "Investimento adicionado",
        type: "success",
        duration: 2000,
      });
    } catch (err) {
      toaster.create({
        title: "Falha ao adicionar investimento",
        description: err instanceof Error ? err.message : "Algo deu errado",
        type: "error",
        duration: 3000,
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteInvestment = async (investmentID: string) => {
    if (!confirm("Deletar esse investimento?")) return;
    try {
      await api.delete(`/investments/${investmentID}`);
      setPortfolio((prev) =>
        prev
          ? {
              ...prev,
              investments: prev.investments.filter((i) => i.id !== investmentID),
            }
          : prev,
      );
      toaster.create({
        title: "Investimento deletado",
        type: "success",
        duration: 2000,
      });
    } catch {
      toaster.create({
        title: "Falha ao deletar investimento",
        type: "error",
        duration: 3000,
      });
    }
  };

  if (loading || !portfolio) {
    return (
      <Center h="50vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  const totalInvested = portfolio.investments.reduce(
    (s, i) => s + (Number(i.amount_invested) || 0),
    0,
  );
  const holdings = portfolio.investments.length;
  const isReal = portfolio.type === "real";

  // Alocação por classe de ativo deste portfolio
  const allocByType = portfolio.investments.reduce<Record<string, number>>(
    (acc, i) => {
      acc[i.asset_type] =
        (acc[i.asset_type] ?? 0) + (Number(i.amount_invested) || 0);
      return acc;
    },
    {},
  );
  const allocSlices: AllocationSlice[] = Object.entries(allocByType)
    .map(([asset_type, value]) => ({ asset_type, value }))
    .sort((a, b) => b.value - a.value);

  // Gradiente do hero por tipo: real=brand cyan, simulado=violeta
  const heroGradient = isReal
    ? "linear-gradient(135deg, rgba(14,165,233,0.28) 0%, rgba(14,165,233,0.10) 50%, #1f2937 100%)"
    : "linear-gradient(135deg, rgba(168,85,247,0.28) 0%, rgba(168,85,247,0.10) 50%, #1f2937 100%)";
  const heroGlow = isReal ? "rgba(14,165,233,0.32)" : "rgba(168,85,247,0.32)";

  return (
    <Stack gap="6">
      <NextLink href="/dashboard/portfolios">
        <Text
          fontSize="sm"
          color="gray.500"
          _hover={{ color: "brand.400" }}
          display="inline-block"
        >
          ← Portfólios
        </Text>
      </NextLink>

      {/* Hero card */}
      <Box
        position="relative"
        overflow="hidden"
        borderRadius="xl"
        border="1px solid"
        borderColor="gray.700"
        bg="gray.800"
        style={{ background: heroGradient }}
        p={{ base: "6", md: "8" }}
      >
        <Box
          position="absolute"
          top="-50px"
          right="-50px"
          w="280px"
          h="280px"
          borderRadius="full"
          style={{ background: `radial-gradient(circle, ${heroGlow} 0%, transparent 70%)` }}
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
            <HStack mb="3">
              <Badge
                colorPalette={isReal ? "blue" : "purple"}
                variant={isReal ? "solid" : "outline"}
                size="md"
              >
                {portfolio.type}
              </Badge>
              <Text fontSize="xs" color="gray.400">
                Criado em{" "}
                {new Date(portfolio.created_at).toLocaleDateString("pt-BR")}
              </Text>
            </HStack>
            <Heading size="xl" color="white" lineHeight="1.1" mb="2">
              {portfolio.name}
            </Heading>
            {portfolio.description && (
              <Text color="gray.400" fontSize="sm" maxW="md">
                {portfolio.description}
              </Text>
            )}
          </Box>
          <NextLink href={`/dashboard/portfolios/${portfolio.id}/edit`}>
            <Button size="sm" variant="outline">
              <PencilIcon size={14} />
              <Text ml="2">Editar</Text>
            </Button>
          </NextLink>
        </Flex>

        <Box
          position="relative"
          mt="6"
          pt="6"
          borderTop="1px solid"
          borderColor="rgba(255,255,255,0.08)"
        >
          <SimpleGrid columns={{ base: 2, md: 4 }} gap="6">
            <Box>
              <Text fontSize="xs" color="gray.400" textTransform="uppercase" letterSpacing="0.05em">
                Total investido
              </Text>
              <Heading size="lg" color="white" mt="1">
                {formatBRL(totalInvested)}
              </Heading>
            </Box>
            <Box>
              <Text fontSize="xs" color="gray.400" textTransform="uppercase" letterSpacing="0.05em">
                Posições
              </Text>
              <Heading size="lg" color="white" mt="1">
                {holdings}
              </Heading>
            </Box>
            <Box>
              <Text fontSize="xs" color="gray.400" textTransform="uppercase" letterSpacing="0.05em">
                Classes
              </Text>
              <Heading size="lg" color="white" mt="1">
                {Object.keys(allocByType).length}
              </Heading>
            </Box>
            <Box>
              <Text fontSize="xs" color="gray.400" textTransform="uppercase" letterSpacing="0.05em">
                Ticket médio
              </Text>
              <Heading size="lg" color="white" mt="1">
                {formatBRL(holdings === 0 ? 0 : totalInvested / holdings)}
              </Heading>
            </Box>
          </SimpleGrid>
        </Box>
      </Box>

      {/* Allocation donut */}
      {allocSlices.length > 0 && (
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
                Alocação deste portfolio
              </Heading>
              <Text fontSize="xs" color="gray.500" mt="1">
                Distribuição por classe de ativo
              </Text>
            </Box>
          </Flex>
          <SimpleGrid columns={{ base: 1, md: 2 }} gap="6" alignItems="center">
            <AllocationDonut data={allocSlices} total={totalInvested} />
            <AllocationLegend data={allocSlices} />
          </SimpleGrid>
        </Box>
      )}

      {/* Add investment form */}
      <Box
        bg="gray.800"
        border="1px solid"
        borderColor="gray.700"
        borderRadius="lg"
        overflow="hidden"
      >
        <Box px="5" py="4" borderBottom="1px solid" borderColor="gray.700">
          <Heading size="sm" color="white">
            Adicionar investimento
          </Heading>
          <Text fontSize="xs" color="gray.500" mt="1">
            Inclua uma nova posição neste portfólio
          </Text>
        </Box>
        <Box p="5">
          <form onSubmit={handleAddInvestment}>
            <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap="4">
              <FieldRoot required>
                <FieldLabel>Ticker</FieldLabel>
                <TickerAutocomplete
                  value={ticker}
                  onChange={(t, asset) => {
                    setTicker(t);
                    // auto-fill asset_type when picking from dropdown,
                    // unless the user already overrode it manually
                    if (asset && !assetTypeTouched) {
                      setAssetType(asset.asset_type);
                    }
                  }}
                  assetType={assetTypeTouched ? assetType : undefined}
                  placeholder="VALE3, AAPL, BTC..."
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Tipo de ativo</FieldLabel>
                <NativeSelectRoot>
                  <NativeSelectField
                    value={assetType}
                    onChange={(e) => {
                      setAssetType(e.target.value as AssetType);
                      setAssetTypeTouched(true);
                    }}
                  >
                    {ASSET_TYPES.map((t) => (
                      <option key={t} value={t}>
                        {ASSET_LABEL[t] ?? t}
                      </option>
                    ))}
                  </NativeSelectField>
                </NativeSelectRoot>
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Valor investido (R$)</FieldLabel>
                <Input
                  value={amountInvested}
                  onChange={(e) => setAmountInvested(e.target.value)}
                  placeholder="1000.00"
                  inputMode="decimal"
                />
              </FieldRoot>
              <FieldRoot>
                <FieldLabel>Quantidade</FieldLabel>
                <Input
                  value={quantity}
                  onChange={(e) => setQuantity(e.target.value)}
                  placeholder="opcional"
                  inputMode="decimal"
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Data de compra</FieldLabel>
                <Input
                  type="date"
                  value={purchaseDate}
                  onChange={(e) => setPurchaseDate(e.target.value)}
                />
              </FieldRoot>
              <FieldRoot>
                <FieldLabel>Notas</FieldLabel>
                <Textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="opcional"
                  rows={1}
                />
              </FieldRoot>
            </SimpleGrid>
            <Button
              type="submit"
              colorPalette="blue"
              mt="5"
              loading={submitting}
            >
              <PlusIcon size={16} />
              <Text ml="2">Adicionar</Text>
            </Button>
          </form>
        </Box>
      </Box>

      {/* Investments table */}
      <Box
        bg="gray.800"
        border="1px solid"
        borderColor="gray.700"
        borderRadius="lg"
        overflow="hidden"
      >
        <Flex
          align="center"
          justify="space-between"
          px="5"
          py="4"
          borderBottom="1px solid"
          borderColor="gray.700"
        >
          <Box>
            <Heading size="sm" color="white">
              Investimentos
            </Heading>
            <Text fontSize="xs" color="gray.500" mt="1">
              Posições ordenadas por data de compra
            </Text>
          </Box>
          <Badge variant="subtle" colorPalette="gray">
            {portfolio.investments.length}
          </Badge>
        </Flex>
        {portfolio.investments.length === 0 ? (
          <Box p="10" textAlign="center">
            <Text color="gray.400">
              Nenhum investimento ainda. Adicione um acima.
            </Text>
          </Box>
        ) : (
          <Box overflowX="auto">
            <Table.Root size="sm" variant="line">
              <Table.Header>
                <Table.Row>
                  <Table.ColumnHeader>Ticker</Table.ColumnHeader>
                  <Table.ColumnHeader>Tipo</Table.ColumnHeader>
                  <Table.ColumnHeader>Preço</Table.ColumnHeader>
                  <Table.ColumnHeader>Valor (R$)</Table.ColumnHeader>
                  <Table.ColumnHeader>Quantidade</Table.ColumnHeader>
                  <Table.ColumnHeader>Compra</Table.ColumnHeader>
                  <Table.ColumnHeader>Notas</Table.ColumnHeader>
                  <Table.ColumnHeader />
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {portfolio.investments.map((inv) => (
                  <Table.Row key={inv.id}>
                    <Table.Cell fontWeight="bold" color="white">
                      <Flex align="center" gap="2">
                        <Box
                          w="8px"
                          h="8px"
                          borderRadius="full"
                          bg={ASSET_COLORS[inv.asset_type] ?? "#6b7280"}
                        />
                        {inv.ticker}
                      </Flex>
                    </Table.Cell>
                    <Table.Cell>
                      <Text fontSize="xs" color="gray.300">
                        {ASSET_LABEL[inv.asset_type] ?? inv.asset_type}
                      </Text>
                    </Table.Cell>
                    <Table.Cell>
                      <PriceBadge
                        ticker={inv.ticker}
                        assetType={inv.asset_type}
                      />
                    </Table.Cell>
                    <Table.Cell color="gain" fontWeight="bold">
                      {formatAmount(inv.amount_invested)}
                    </Table.Cell>
                    <Table.Cell color="gray.300">
                      {formatQuantity(inv.quantity)}
                    </Table.Cell>
                    <Table.Cell color="gray.400">
                      {new Date(inv.purchase_date).toLocaleDateString("pt-BR")}
                    </Table.Cell>
                    <Table.Cell color="gray.500" maxW="200px" truncate>
                      {inv.notes ?? "—"}
                    </Table.Cell>
                    <Table.Cell>
                      <Button
                        size="xs"
                        variant="ghost"
                        colorPalette="red"
                        onClick={() => handleDeleteInvestment(inv.id)}
                        aria-label="Deletar"
                      >
                        <TrashIcon size={14} />
                      </Button>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table.Root>
          </Box>
        )}
      </Box>
    </Stack>
  );
}
