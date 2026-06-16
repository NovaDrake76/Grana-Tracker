"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Box,
  Button,
  FieldLabel,
  FieldRoot,
  Flex,
  Grid,
  Heading,
  Input,
  Text,
  VStack,
  Link as ChakraLink,
} from "@chakra-ui/react";
import NextLink from "next/link";
import { useAuth } from "@/context/AuthContext";
import { toaster } from "@/lib/toaster";
import { AuthHero } from "@/components/AuthHero";

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
    <Grid templateColumns={{ base: "1fr", lg: "1fr 1fr" }} minH="100vh">
      <AuthHero />

      <Flex
        direction="column"
        align="center"
        justify="center"
        bg="gray.950"
        p={{ base: "6", md: "12" }}
        position="relative"
      >
        <Box w="100%" maxW="420px">
          <Box mb="8">
            <Heading size="lg" color="white" mb="2">
              Criar conta
            </Heading>
            <Text color="gray.400" fontSize="sm">
              Comece a acompanhar suas carteiras em poucos segundos
            </Text>
          </Box>

          <form onSubmit={handleSubmit}>
            <VStack gap="5" align="stretch">
              <FieldRoot required>
                <FieldLabel>Nome</FieldLabel>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Seu nome"
                  bg="gray.800"
                  borderColor="gray.700"
                  _hover={{ borderColor: "gray.600" }}
                  _focus={{ borderColor: "brand.500" }}
                  size="lg"
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Email</FieldLabel>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="seu@email.com"
                  bg="gray.800"
                  borderColor="gray.700"
                  _hover={{ borderColor: "gray.600" }}
                  _focus={{ borderColor: "brand.500" }}
                  size="lg"
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Senha</FieldLabel>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Mínimo de 8 caracteres"
                  bg="gray.800"
                  borderColor="gray.700"
                  _hover={{ borderColor: "gray.600" }}
                  _focus={{ borderColor: "brand.500" }}
                  size="lg"
                />
              </FieldRoot>
              <Button
                type="submit"
                colorPalette="blue"
                w="100%"
                loading={loading}
                size="lg"
              >
                Criar conta
              </Button>
            </VStack>
          </form>

          <Box
            mt="6"
            pt="6"
            borderTop="1px solid"
            borderColor="gray.800"
            textAlign="center"
          >
            <Text fontSize="sm" color="gray.400">
              Já tem uma conta?{" "}
              <ChakraLink asChild color="brand.400" fontWeight="medium">
                <NextLink href="/login">Entrar</NextLink>
              </ChakraLink>
            </Text>
          </Box>
        </Box>
      </Flex>
    </Grid>
  );
}
