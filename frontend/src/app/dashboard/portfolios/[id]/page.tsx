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

const ASSET_TYPES: AssetType[] = ["stock", "crypto", "etf", "index"];

function formatAmount(value: string) {
  if (!value) return "—";
  const n = Number(value);
  if (Number.isNaN(n)) return value;
  return n.toLocaleString(undefined, {
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

  const [portfolio, setPortfolio] = useState<PortfolioWithInvestments | null>(null);
  const [loading, setLoading] = useState(true);

  // add-investment form state
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
        title: "Portfolio not found",
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
        title: "Investment added",
        type: "success",
        duration: 2000,
      });
    } catch (err) {
      toaster.create({
        title: "Failed to add investment",
        description: err instanceof Error ? err.message : "Something went wrong",
        type: "error",
        duration: 3000,
      });
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteInvestment = async (investmentID: string) => {
    if (!confirm("Delete this investment?")) return;
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
        title: "Investment deleted",
        type: "success",
        duration: 2000,
      });
    } catch {
      toaster.create({
        title: "Failed to delete investment",
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

  return (
    <Stack gap="6">
      <Flex justify="space-between" align="center">
        <Box>
          <HStack mb="2">
            <Heading size="lg">{portfolio.name}</Heading>
            <Badge
              colorPalette={portfolio.type === "real" ? "blue" : "purple"}
              variant={portfolio.type === "real" ? "solid" : "outline"}
            >
              {portfolio.type}
            </Badge>
          </HStack>
          {portfolio.description && (
            <Text color="gray.400">{portfolio.description}</Text>
          )}
        </Box>
        <HStack>
          <NextLink href={`/dashboard/portfolios/${portfolio.id}/edit`}>
            <Button size="sm" variant="outline">
              Edit portfolio
            </Button>
          </NextLink>
          <NextLink href="/dashboard/portfolios">
            <Button size="sm" variant="ghost">
              Back
            </Button>
          </NextLink>
        </HStack>
      </Flex>

      <Box bg="gray.800" p="5" borderRadius="lg">
        <Heading size="md" mb="4">
          Add investment
        </Heading>
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
              <FieldLabel>Asset type</FieldLabel>
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
              <FieldLabel>Amount invested</FieldLabel>
              <Input
                value={amountInvested}
                onChange={(e) => setAmountInvested(e.target.value)}
                placeholder="1000.00"
                inputMode="decimal"
              />
            </FieldRoot>
            <FieldRoot>
              <FieldLabel>Quantity</FieldLabel>
              <Input
                value={quantity}
                onChange={(e) => setQuantity(e.target.value)}
                placeholder="optional"
                inputMode="decimal"
              />
            </FieldRoot>
            <FieldRoot required>
              <FieldLabel>Purchase date</FieldLabel>
              <Input
                type="date"
                value={purchaseDate}
                onChange={(e) => setPurchaseDate(e.target.value)}
              />
            </FieldRoot>
            <FieldRoot>
              <FieldLabel>Notes</FieldLabel>
              <Textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="optional"
                rows={1}
              />
            </FieldRoot>
          </SimpleGrid>
          <Button
            type="submit"
            colorPalette="blue"
            mt="4"
            loading={submitting}
          >
            Add investment
          </Button>
        </form>
      </Box>

      <Box bg="gray.800" p="5" borderRadius="lg">
        <Heading size="md" mb="4">
          Investments ({portfolio.investments.length})
        </Heading>
        {portfolio.investments.length === 0 ? (
          <Text color="gray.400">No investments yet. Add one above.</Text>
        ) : (
          <Table.Root size="sm" variant="line">
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeader>Ticker</Table.ColumnHeader>
                <Table.ColumnHeader>Type</Table.ColumnHeader>
                <Table.ColumnHeader>Amount invested</Table.ColumnHeader>
                <Table.ColumnHeader>Quantity</Table.ColumnHeader>
                <Table.ColumnHeader>Purchase date</Table.ColumnHeader>
                <Table.ColumnHeader>Notes</Table.ColumnHeader>
                <Table.ColumnHeader />
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {portfolio.investments.map((inv) => (
                <Table.Row key={inv.id}>
                  <Table.Cell fontWeight="medium">{inv.ticker}</Table.Cell>
                  <Table.Cell>
                    <Badge variant="outline">{inv.asset_type}</Badge>
                  </Table.Cell>
                  <Table.Cell>{formatAmount(inv.amount_invested)}</Table.Cell>
                  <Table.Cell>{formatQuantity(inv.quantity)}</Table.Cell>
                  <Table.Cell>{inv.purchase_date}</Table.Cell>
                  <Table.Cell color="gray.400">
                    {inv.notes ?? "—"}
                  </Table.Cell>
                  <Table.Cell>
                    <Button
                      size="xs"
                      variant="outline"
                      colorPalette="red"
                      onClick={() => handleDeleteInvestment(inv.id)}
                    >
                      Delete
                    </Button>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table.Root>
        )}
      </Box>
    </Stack>
  );
}
