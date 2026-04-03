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
} from "@chakra-ui/react";
import NextLink from "next/link";
import { api } from "@/lib/api";
import { toaster } from "@/lib/toaster";
import type { Portfolio, ApiResponse } from "@/types";

export default function PortfoliosPage() {
  const [portfolios, setPortfolios] = useState<Portfolio[]>([]);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  const fetchPortfolios = useCallback(async () => {
    try {
      const res = await api.get<ApiResponse<Portfolio[]>>("/portfolios");
      setPortfolios(res.data);
    } catch {
      toaster.create({
        title: "Failed to load portfolios",
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
    if (!confirm("Are you sure you want to delete this portfolio?")) return;

    try {
      await api.delete(`/portfolios/${id}`);
      setPortfolios((prev) => prev.filter((p) => p.id !== id));
      toaster.create({
        title: "Portfolio deleted",
        type: "success",
        duration: 2000,
      });
    } catch {
      toaster.create({
        title: "Failed to delete portfolio",
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

  return (
    <Box>
      <Flex justify="space-between" align="center" mb="6">
        <Heading size="lg">Portfolios</Heading>
        <NextLink href="/dashboard/portfolios/new">
          <Button colorPalette="blue" size="sm">
            New Portfolio
          </Button>
        </NextLink>
      </Flex>

      {portfolios.length === 0 ? (
        <Center h="40vh" flexDirection="column">
          <Text color="gray.400" mb="4">
            No portfolios yet
          </Text>
          <NextLink href="/dashboard/portfolios/new">
            <Button colorPalette="blue">Create your first portfolio</Button>
          </NextLink>
        </Center>
      ) : (
        <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap="4">
          {portfolios.map((portfolio) => (
            <Box
              key={portfolio.id}
              bg="gray.800"
              p="5"
              borderRadius="lg"
              border="1px solid"
              borderColor="gray.700"
              _hover={{ borderColor: "gray.600" }}
              transition="border-color 0.2s"
            >
              <Flex justify="space-between" align="start" mb="3">
                <Heading size="sm">{portfolio.name}</Heading>
                <Badge
                  colorPalette={portfolio.type === "real" ? "blue" : "purple"}
                  variant={portfolio.type === "real" ? "solid" : "outline"}
                >
                  {portfolio.type}
                </Badge>
              </Flex>

              {portfolio.description && (
                <Text fontSize="sm" color="gray.400" mb="3" lineClamp={2}>
                  {portfolio.description}
                </Text>
              )}

              <Text fontSize="xs" color="gray.500" mb="3">
                Created {new Date(portfolio.created_at).toLocaleDateString()}
              </Text>

              <HStack gap="2">
                <Button
                  size="xs"
                  variant="outline"
                  onClick={() =>
                    router.push(`/dashboard/portfolios/${portfolio.id}/edit`)
                  }
                >
                  Edit
                </Button>
                <Button
                  size="xs"
                  variant="outline"
                  colorPalette="red"
                  onClick={() => handleDelete(portfolio.id)}
                >
                  Delete
                </Button>
              </HStack>
            </Box>
          ))}
        </SimpleGrid>
      )}
    </Box>
  );
}
