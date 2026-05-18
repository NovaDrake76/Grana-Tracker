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
import { StatCard } from "@/components/StatCard";
import {
  LayersIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
  WalletIcon,
  TrendingUpIcon,
  SparkleIcon,
} from "@/components/Icons";

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

const assetTypeColor: Record<AssetType, string> = {
  stock: "blue",
  crypto: "orange",
  etf: "purple",
  index: "green",
};

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
  const assetBreakdown = portfolio.investments.reduce<Record<string, number>>(
    (acc, i) => {
      acc[i.asset_type] = (acc[i.asset_type] ?? 0) + 1;
      return acc;
    },
    {},
  );
  const topAssetType = Object.entries(assetBreakdown).sort(
    (a, b) => b[1] - a[1],
  )[0];

  const isReal = portfolio.type === "real";

  return (
    <Stack gap="6">
      <NextLink href="/dashboard/portfolios">
        <Text
          fontSize="sm"
          color="gray.500"
          _hover={{ color: "brand.300" }}
          display="inline-block"
          transition="color 0.15s"
        >
          ← Portfólios
        </Text>
      </NextLink>

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
            <HStack mb="3">
              <Badge
                colorPalette={isReal ? "blue" : "purple"}
                variant={isReal ? "solid" : "outline"}
                size="sm"
              >
                {portfolio.type}
              </Badge>
              <Text fontSize="sm" color="gray.500">
                Criado em{" "}
                {new Date(portfolio.created_at).toLocaleDateString("pt-BR")}
              </Text>
            </HStack>
            <Heading
              size="2xl"
              className="gradient-text"
              lineHeight="1.1"
              mb="2"
            >
              {portfolio.name}
            </Heading>
            {portfolio.description && (
              <Text color="gray.300" maxW="2xl">
                {portfolio.description}
              </Text>
            )}
          </Box>
          <NextLink href={`/dashboard/portfolios/${portfolio.id}/edit`}>
            <Button
              size="sm"
              variant="outline"
              borderColor="rgba(148, 163, 184, 0.3)"
              _hover={{
                bg: "rgba(148, 163, 184, 0.1)",
                borderColor: "brand.400",
              }}
            >
              <PencilIcon size={14} />
              <Text ml="2">Editar</Text>
            </Button>
          </NextLink>
        </Flex>
      </Box>

      <SimpleGrid columns={{ base: 1, sm: 3 }} gap="4">
        <StatCard
          label="Total investido"
          value={formatBRL(totalInvested)}
          helper={`${holdings} posição${holdings === 1 ? "" : "ões"}`}
          icon={<WalletIcon size={16} />}
          accent="gain"
        />
        <StatCard
          label="Posições"
          value={holdings}
          helper={
            holdings === 0
              ? "Nenhuma ainda"
              : `${Object.keys(assetBreakdown).length} classe${Object.keys(assetBreakdown).length === 1 ? "" : "s"} de ativo`
          }
          icon={<LayersIcon size={16} />}
          accent="brand"
        />
        <StatCard
          label="Classe principal"
          value={topAssetType ? topAssetType[0] : "—"}
          helper={
            topAssetType
              ? `${topAssetType[1]} posição${topAssetType[1] === 1 ? "" : "ões"}`
              : "Adicione um investimento"
          }
          icon={<TrendingUpIcon size={16} />}
          accent="purple"
        />
      </SimpleGrid>

      <Box className="glass-card" borderRadius="xl" overflow="hidden">
        <Flex
          align="center"
          gap="3"
          px="5"
          py="4"
          borderBottom="1px solid rgba(148, 163, 184, 0.08)"
        >
          <Flex
            w="36px"
            h="36px"
            align="center"
            justify="center"
            color="brand.300"
            borderRadius="lg"
            style={{
              background:
                "linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(14, 165, 233, 0.05))",
              border: "1px solid rgba(14, 165, 233, 0.25)",
            }}
          >
            <PlusIcon size={16} />
          </Flex>
          <Box>
            <Heading size="sm" color="white">
              Adicionar investimento
            </Heading>
            <Text fontSize="xs" color="gray.500">
              Inclua uma nova posição neste portfólio
            </Text>
          </Box>
        </Flex>
        <Box p="5">
          <form onSubmit={handleAddInvestment}>
            <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap="4">
              <FieldRoot required>
                <FieldLabel>Ticker</FieldLabel>
                <Input
                  value={ticker}
                  onChange={(e) => setTicker(e.target.value)}
                  placeholder="AAPL"
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Tipo de ativo</FieldLabel>
                <NativeSelectRoot>
                  <NativeSelectField
                    value={assetType}
                    onChange={(e) => setAssetType(e.target.value as AssetType)}
                  >
                    {ASSET_TYPES.map((t) => (
                      <option key={t} value={t}>
                        {t}
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
              style={{
                background: "linear-gradient(135deg, #0ea5e9, #0284c7)",
                boxShadow: "0 8px 24px -8px rgba(14, 165, 233, 0.5)",
              }}
            >
              <PlusIcon size={16} />
              <Text ml="2">Adicionar investimento</Text>
            </Button>
          </form>
        </Box>
      </Box>

      <Box className="glass-card" borderRadius="xl" overflow="hidden">
        <Flex
          align="center"
          justify="space-between"
          px="5"
          py="4"
          borderBottom="1px solid rgba(148, 163, 184, 0.08)"
        >
          <Heading size="sm" color="white">
            Investimentos
          </Heading>
          <Badge variant="subtle" colorPalette="gray">
            {portfolio.investments.length}
          </Badge>
        </Flex>
        {portfolio.investments.length === 0 ? (
          <Box p="10" textAlign="center">
            <Flex
              w="56px"
              h="56px"
              mx="auto"
              mb="4"
              align="center"
              justify="center"
              color="brand.300"
              borderRadius="full"
              style={{
                background:
                  "linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(168, 85, 247, 0.15))",
                boxShadow: "0 0 30px -8px rgba(14, 165, 233, 0.4)",
              }}
            >
              <SparkleIcon size={28} />
            </Flex>
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
                      {inv.ticker}
                    </Table.Cell>
                    <Table.Cell>
                      <Badge
                        size="sm"
                        variant="subtle"
                        colorPalette={assetTypeColor[inv.asset_type]}
                      >
                        {inv.asset_type}
                      </Badge>
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
