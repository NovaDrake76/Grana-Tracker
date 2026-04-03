"use client";

import { useState } from "react";
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
} from "@chakra-ui/react";
import { api } from "@/lib/api";
import { toaster } from "@/lib/toaster";

export default function NewPortfolioPage() {
  const [name, setName] = useState("");
  const [type, setType] = useState("real");
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await api.post("/portfolios", {
        name,
        type,
        description: description || null,
      });
      toaster.create({
        title: "Portfolio created",
        type: "success",
        duration: 2000,
      });
      router.push("/dashboard/portfolios");
    } catch (err) {
      toaster.create({
        title: "Failed to create portfolio",
        description: err instanceof Error ? err.message : "Something went wrong",
        type: "error",
        duration: 3000,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box maxW="lg">
      <Heading size="lg" mb="6">
        New Portfolio
      </Heading>
      <Box bg="gray.800" p="6" borderRadius="lg">
        <form onSubmit={handleSubmit}>
          <VStack gap="4">
            <FieldRoot required>
              <FieldLabel>Name</FieldLabel>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My Portfolio"
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
                placeholder="Optional description"
              />
            </FieldRoot>
            <Button
              type="submit"
              colorPalette="blue"
              w="100%"
              loading={loading}
            >
              Create Portfolio
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
