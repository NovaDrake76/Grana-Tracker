"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Box,
  Button,
  Container,
  FieldLabel,
  FieldRoot,
  Flex,
  Heading,
  Input,
  Text,
  VStack,
  Link as ChakraLink,
} from "@chakra-ui/react";
import NextLink from "next/link";
import { useAuth } from "@/context/AuthContext";
import { toaster } from "@/lib/toaster";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const { login, isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace("/dashboard");
    }
  }, [isAuthenticated, isLoading, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await login(email, password);
      router.push("/dashboard");
    } catch (err) {
      toaster.create({
        title: "Falha no login",
        description: err instanceof Error ? err.message : "Algo deu errado",
        type: "error",
        duration: 3000,
      });
    } finally {
      setLoading(false);
    }
  };

  if (isLoading) return null;

  return (
    <Box minH="100vh" bg="gray.900">
      <Container maxW="md" py={{ base: "10", md: "20" }}>
        <VStack gap="6">
          <Flex direction="column" align="center" gap="2">
            <Flex
              w="48px"
              h="48px"
              align="center"
              justify="center"
              bg="brand.600"
              color="white"
              borderRadius="md"
              fontSize="2xl"
              fontWeight="bold"
            >
              G
            </Flex>
            <Heading size="lg" color="white">
              Grana Tracker
            </Heading>
            <Text color="gray.400" textAlign="center" fontSize="sm">
              Acompanhe investimentos reais e simulados em um só lugar
            </Text>
          </Flex>

          <Box
            w="100%"
            bg="gray.800"
            p="8"
            borderRadius="md"
            border="1px solid"
            borderColor="gray.700"
          >
            <form onSubmit={handleSubmit}>
              <VStack gap="5">
                <Heading size="md" color="white" alignSelf="start">
                  Entrar
                </Heading>
                <FieldRoot required>
                  <FieldLabel>Email</FieldLabel>
                  <Input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="seu@email.com"
                  />
                </FieldRoot>
                <FieldRoot required>
                  <FieldLabel>Senha</FieldLabel>
                  <Input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                  />
                </FieldRoot>
                <Button
                  type="submit"
                  colorPalette="blue"
                  w="100%"
                  loading={loading}
                >
                  Entrar
                </Button>
                <Box
                  w="100%"
                  borderTop="1px solid"
                  borderColor="gray.700"
                  pt="4"
                >
                  <Text fontSize="sm" color="gray.400" textAlign="center">
                    Não tem uma conta?{" "}
                    <ChakraLink asChild color="brand.400" fontWeight="medium">
                      <NextLink href="/register">Criar conta</NextLink>
                    </ChakraLink>
                  </Text>
                </Box>
              </VStack>
            </form>
          </Box>

          <Text fontSize="xs" color="gray.600" textAlign="center">
            DIM0547 · Desenvolvimento de Sistemas Web II com Go · UFRN 2026.1
          </Text>
        </VStack>
      </Container>
    </Box>
  );
}
