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
  PortfolioIcon,
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
      <Box className="hero-card" p={{ base: "6", md: "7" }}>
        <Flex
          justify="space-between"
          align={{ base: "start", md: "center" }}
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
              SEUS PORTFÓLIOS
            </Text>
            <Heading size="xl" className="gradient-text">
              Carteiras
            </Heading>
            <Text color="gray.400" mt="2">
              Reais e simuladas, lado a lado
            </Text>
          </Box>
          <NextLink href="/dashboard/portfolios/new">
            <Button
              colorPalette="blue"
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

      {portfolios.length === 0 ? (
        <Box className="glass-card" borderRadius="xl" p="12" textAlign="center">
          <Flex
            w="72px"
            h="72px"
            mx="auto"
            mb="5"
            align="center"
            justify="center"
            color="brand.300"
            borderRadius="full"
            style={{
              background:
                "linear-gradient(135deg, rgba(14, 165, 233, 0.2), rgba(168, 85, 247, 0.15))",
              boxShadow: "0 0 40px -8px rgba(14, 165, 233, 0.4)",
            }}
          >
            <PortfolioIcon size={36} />
          </Flex>
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
              className="glass-card lift"
              borderRadius="xl"
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
