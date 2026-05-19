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
import type { Portfolio, ApiResponse } from "@/types";
import {
  EyeIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from "@/components/Icons";

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

      {portfolios.length === 0 ? (
        <Box
          bg="gray.800"
          border="1px solid"
          borderColor="gray.700"
          borderRadius="md"
          p="10"
          textAlign="center"
        >
          <Heading size="md" color="white" mb="2">
            Nenhum portfólio ainda
          </Heading>
          <Text color="gray.400" mb="5">
            Crie sua primeira carteira para começar a acompanhar.
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
          {portfolios.map((portfolio) => (
            <Box
              key={portfolio.id}
              className="lift"
              bg="gray.800"
              borderRadius="md"
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
                  >
                    {portfolio.type}
                  </Badge>
                </Flex>

                {portfolio.description ? (
                  <Text
                    fontSize="sm"
                    color="gray.400"
                    mb="4"
                    lineClamp={2}
                    minH="42px"
                  >
                    {portfolio.description}
                  </Text>
                ) : (
                  <Text
                    fontSize="sm"
                    color="gray.600"
                    mb="4"
                    fontStyle="italic"
                    minH="42px"
                  >
                    Sem descrição
                  </Text>
                )}

                <Text fontSize="xs" color="gray.500" mb="4">
                  Criado em{" "}
                  {new Date(portfolio.created_at).toLocaleDateString("pt-BR")}
                </Text>

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
          ))}
        </SimpleGrid>
      )}
    </Stack>
  );
}
