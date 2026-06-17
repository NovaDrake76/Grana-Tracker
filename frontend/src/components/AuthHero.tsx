"use client";

import { Box, Flex, Heading, Stack, Text } from "@chakra-ui/react";
import {
  LayersIcon,
  TrendingUpIcon,
  WalletIcon,
} from "@/components/Icons";
import type { ReactNode } from "react";

interface Feature {
  icon: ReactNode;
  title: string;
  desc: string;
}

const FEATURES: Feature[] = [
  {
    icon: <WalletIcon size={18} />,
    title: "Carteiras reais e simuladas",
    desc: "Separe o que você de fato investe do que está testando.",
  },
  {
    icon: <LayersIcon size={18} />,
    title: "Ações, ETFs e cripto",
    desc: "Tudo no mesmo lugar, com alocação por classe.",
  },
  {
    icon: <TrendingUpIcon size={18} />,
    title: "Visão consolidada",
    desc: "Dashboard com totais, distribuição e histórico.",
  },
];

export function AuthHero() {
  return (
    <Box
      position="relative"
      display={{ base: "none", lg: "flex" }}
      flexDirection="column"
      h="100vh"
      w="100%"
      p="12"
      bg="gray.900"
      overflow="hidden"
    >
      {/* Background gradients (radial overlays) */}
      <Box
        position="absolute"
        top="-100px"
        left="-120px"
        w="500px"
        h="500px"
        borderRadius="full"
        style={{ background: "radial-gradient(circle, rgba(14,165,233,0.35) 0%, transparent 70%)" }}
        pointerEvents="none"
      />
      <Box
        position="absolute"
        bottom="-150px"
        right="-100px"
        w="450px"
        h="450px"
        borderRadius="full"
        style={{ background: "radial-gradient(circle, rgba(168,85,247,0.28) 0%, transparent 70%)" }}
        pointerEvents="none"
      />

      {/* Logo + brand */}
      <Flex align="center" gap="3" position="relative" mb="12">
        <Flex
          w="44px"
          h="44px"
          align="center"
          justify="center"
          bg="brand.600"
          color="white"
          borderRadius="md"
          fontWeight="bold"
          fontSize="xl"
          boxShadow="0 8px 24px rgba(14,165,233,0.35)"
        >
          G
        </Flex>
        <Box>
          <Heading size="md" color="white" lineHeight="1">
            Grana Tracker
          </Heading>
          <Text fontSize="xs" color="gray.400">
            DIM0547 · UFRN 2026.1
          </Text>
        </Box>
      </Flex>

      {/* Headline */}
      <Box position="relative" maxW="md" mb="10">
        <Heading
          size="2xl"
          color="white"
          lineHeight="1.1"
          letterSpacing="-0.02em"
          mb="4"
        >
          Onde seus investimentos finalmente fazem sentido.
        </Heading>
        <Text color="gray.400" fontSize="md">
          Uma plataforma simples pra acompanhar carteiras reais e simuladas,
          de ações a cripto, com totais e alocação na palma da mão.
        </Text>
      </Box>

      {/* Feature list */}
      <Stack gap="4" position="relative" maxW="md">
        {FEATURES.map((f) => (
          <Flex key={f.title} gap="3" align="start">
            <Flex
              w="36px"
              h="36px"
              align="center"
              justify="center"
              borderRadius="md"
              bg="rgba(14,165,233,0.12)"
              color="brand.300"
              flexShrink="0"
            >
              {f.icon}
            </Flex>
            <Box>
              <Text color="white" fontWeight="medium" fontSize="sm">
                {f.title}
              </Text>
              <Text color="gray.400" fontSize="xs" mt="0.5">
                {f.desc}
              </Text>
            </Box>
          </Flex>
        ))}
      </Stack>

      {/* Footer credit */}
      <Box mt="auto" position="relative">
        <Text fontSize="xs" color="gray.600">
          Projeto acadêmico · Time DIM0547 — Breno · Nathan · Heittor
        </Text>
      </Box>
    </Box>
  );
}
