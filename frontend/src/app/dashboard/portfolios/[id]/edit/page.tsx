"use client";

import { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import {
  Box,
  Button,
  FieldLabel,
  FieldRoot,
  Heading,
  Input,
  NativeSelectField,
  NativeSelectRoot,
  Textarea,
  VStack,
  Spinner,
  Center,
} from "@chakra-ui/react";
import { api } from "@/lib/api";
import { toaster } from "@/lib/toaster";
import type { Portfolio, ApiResponse } from "@/types";

export default function EditPortfolioPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [name, setName] = useState("");
  const [type, setType] = useState("real");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const router = useRouter();

  useEffect(() => {
    async function fetchPortfolio() {
      try {
        const res = await api.get<ApiResponse<Portfolio>>(`/portfolios/${id}`);
        setName(res.data.name);
        setType(res.data.type);
        setDescription(res.data.description || "");
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
    }
    fetchPortfolio();
  }, [id, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.put(`/portfolios/${id}`, {
        name,
        type,
        description: description || null,
      });
      toaster.create({
        title: "Portfolio updated",
        type: "success",
        duration: 2000,
      });
      router.push("/dashboard/portfolios");
    } catch (err) {
      toaster.create({
        title: "Failed to update portfolio",
        description: err instanceof Error ? err.message : "Something went wrong",
        type: "error",
        duration: 3000,
      });
    } finally {
      setSaving(false);
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
    <Box maxW="lg">
      <Heading size="lg" mb="6">
        Edit Portfolio
      </Heading>
      <Box bg="gray.800" p="6" borderRadius="lg">
        <form onSubmit={handleSubmit}>
          <VStack gap="4">
            <FieldRoot required>
              <FieldLabel>Name</FieldLabel>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </FieldRoot>
            <FieldRoot required>
              <FieldLabel>Type</FieldLabel>
              <NativeSelectRoot>
                <NativeSelectField
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                >
                  <option value="real">Real</option>
                  <option value="simulated">Simulated</option>
                </NativeSelectField>
              </NativeSelectRoot>
            </FieldRoot>
            <FieldRoot>
              <FieldLabel>Description</FieldLabel>
              <Textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </FieldRoot>
            <Button
              type="submit"
              colorPalette="blue"
              w="100%"
              loading={saving}
            >
              Save Changes
            </Button>
            <Button variant="ghost" w="100%" onClick={() => router.back()}>
              Cancel
            </Button>
          </VStack>
        </form>
      </Box>
    </Box>
  );
}
