"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Box,
  Button,
  Container,
  FieldLabel,
  FieldRoot,
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
      router.replace("/dashboard/portfolios");
    }
  }, [isAuthenticated, isLoading, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await register(name, email, password);
      router.push("/dashboard/portfolios");
    } catch (err) {
      toaster.create({
        title: "Registration failed",
        description: err instanceof Error ? err.message : "Something went wrong",
        type: "error",
        duration: 3000,
      });
    } finally {
      setLoading(false);
    }
  };

  if (isLoading) return null;

  return (
    <Container maxW="md" py="20">
      <VStack gap="8">
        <Heading size="xl" color="brand.500">
          Grana Tracker
        </Heading>
        <Box w="100%" bg="gray.800" p="8" borderRadius="lg">
          <form onSubmit={handleSubmit}>
            <VStack gap="4">
              <Heading size="md">Create Account</Heading>
              <FieldRoot required>
                <FieldLabel>Name</FieldLabel>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Your name"
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Email</FieldLabel>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="your@email.com"
                />
              </FieldRoot>
              <FieldRoot required>
                <FieldLabel>Password</FieldLabel>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="At least 6 characters"
                />
              </FieldRoot>
              <Button
                type="submit"
                colorPalette="blue"
                w="100%"
                loading={loading}
              >
                Register
              </Button>
              <Text fontSize="sm">
                Already have an account?{" "}
                <ChakraLink asChild color="brand.500">
                  <NextLink href="/login">Login</NextLink>
                </ChakraLink>
              </Text>
            </VStack>
          </form>
        </Box>
      </VStack>
    </Container>
  );
}
