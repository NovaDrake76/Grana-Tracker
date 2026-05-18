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

export default function RegisterPage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const { register, isAuthenticated, isLoading } = useAuth();
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
      await register(name, email, password);
      router.push("/dashboard");
    } catch (err) {
      toaster.create({
        title: "Falha no cadastro",
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
    <Box className="auth-backdrop">
      <Container maxW="md" py={{ base: "10", md: "20" }} position="relative" zIndex="1">
        <VStack gap="8">
          <Flex direction="column" align="center" gap="3">
            <Flex
              w="64px"
              h="64px"
              align="center"
              justify="center"
              borderRadius="2xl"
              fontSize="3xl"
              fontWeight="bold"
              color="gray.900"
              style={{
                background: "linear-gradient(135deg, #7dd3fc, #0ea5e9 50%, #0369a1)",
                boxShadow:
                  "0 12px 40px -8px rgba(14, 165, 233, 0.6), inset 0 1px 0 0 rgba(255,255,255,0.4)",
              }}
            >
              G
            </Flex>
            <Heading
              size="2xl"
              className="gradient-text"
              textAlign="center"
            >
              Grana Tracker
            </Heading>
            <Text color="gray.400" textAlign="center" fontSize="sm">
              Crie uma conta gratuita para começar
            </Text>
          </Flex>

          <Box w="100%" className="glass-card" p="8" borderRadius="2xl">
            <form onSubmit={handleSubmit}>
              <VStack gap="5">
                <Heading size="md" color="white" alignSelf="start">
                  Criar conta
                </Heading>
                <FieldRoot required>
                  <FieldLabel>Nome</FieldLabel>
                  <Input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Seu nome"
                    bg="rgba(15, 23, 42, 0.5)"
                    borderColor="rgba(148, 163, 184, 0.2)"
                    _hover={{ borderColor: "rgba(148, 163, 184, 0.4)" }}
                    _focus={{ borderColor: "brand.400", boxShadow: "0 0 0 1px var(--brand-500)" }}
                  />
                </FieldRoot>
                <FieldRoot required>
                  <FieldLabel>Email</FieldLabel>
                  <Input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="seu@email.com"
                    bg="rgba(15, 23, 42, 0.5)"
                    borderColor="rgba(148, 163, 184, 0.2)"
                    _hover={{ borderColor: "rgba(148, 163, 184, 0.4)" }}
                    _focus={{ borderColor: "brand.400", boxShadow: "0 0 0 1px var(--brand-500)" }}
                  />
                </FieldRoot>
                <FieldRoot required>
                  <FieldLabel>Senha</FieldLabel>
                  <Input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Mínimo de 6 caracteres"
                    bg="rgba(15, 23, 42, 0.5)"
                    borderColor="rgba(148, 163, 184, 0.2)"
                    _hover={{ borderColor: "rgba(148, 163, 184, 0.4)" }}
                    _focus={{ borderColor: "brand.400", boxShadow: "0 0 0 1px var(--brand-500)" }}
                  />
                </FieldRoot>
                <Button
                  type="submit"
                  colorPalette="blue"
                  w="100%"
                  loading={loading}
                  size="md"
                  style={{
                    background: "linear-gradient(135deg, #0ea5e9, #0284c7)",
                    boxShadow: "0 8px 24px -8px rgba(14, 165, 233, 0.6)",
                  }}
                >
                  Criar conta
                </Button>
                <Box
                  w="100%"
                  borderTop="1px solid rgba(148, 163, 184, 0.1)"
                  pt="4"
                >
                  <Text fontSize="sm" color="gray.400" textAlign="center">
                    Já tem uma conta?{" "}
                    <ChakraLink asChild color="brand.300" fontWeight="medium">
                      <NextLink href="/login">Entrar</NextLink>
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
