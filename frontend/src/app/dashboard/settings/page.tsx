"use client";

import { useEffect, useState } from "react";
import {
  Box,
  Button,
  Flex,
  Heading,
  HStack,
  NativeSelectField,
  NativeSelectRoot,
  Stack,
  Text,
} from "@chakra-ui/react";
import { api } from "@/lib/api";
import { toaster } from "@/lib/toaster";
import { useAuth } from "@/context/AuthContext";
import type { ApiResponse, User } from "@/types";

// US09 — Configurações de Moeda Preferida.
// PUT /user/me com preferred_currency BRL/USD; depois chama refreshUser() do
// AuthContext pra que o sidebar/dashboard reflitam o novo valor sem F5.

type Currency = "BRL" | "USD";

const CURRENCY_LABEL: Record<Currency, string> = {
  BRL: "Real (R$ BRL)",
  USD: "Dólar (US$ USD)",
};

export default function SettingsPage() {
  const { user, refreshUser } = useAuth();

  const initial: Currency =
    user?.preferred_currency?.toUpperCase() === "USD" ? "USD" : "BRL";
  const [selectedCurrency, setSelectedCurrency] = useState<Currency>(initial);
  const [saving, setSaving] = useState(false);

  // Sincroniza quando o user carrega depois do mount.
  useEffect(() => {
    if (user?.preferred_currency) {
      const c = user.preferred_currency.toUpperCase();
      if (c === "BRL" || c === "USD") {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setSelectedCurrency(c);
      }
    }
  }, [user?.preferred_currency]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.put<ApiResponse<User>>("/user/me", {
        preferred_currency: selectedCurrency,
      });
      await refreshUser();
      toaster.create({
        title: "Configurações salvas",
        description: `Moeda preferida: ${selectedCurrency}`,
        type: "success",
        duration: 2500,
      });
    } catch (err) {
      toaster.create({
        title: "Falha ao salvar configurações",
        description: err instanceof Error ? err.message : "Tente novamente",
        type: "error",
        duration: 3000,
      });
    } finally {
      setSaving(false);
    }
  };

  const dirty = selectedCurrency !== initial;

  return (
    <Stack gap="6">
      <Box>
        <Heading size="lg" color="white" mb="1">
          Configurações
        </Heading>
        <Text color="gray.400" fontSize="sm">
          Ajuste preferências da sua conta
        </Text>
      </Box>

      <Box
        bg="gray.800"
        border="1px solid"
        borderColor="gray.700"
        borderRadius="lg"
        overflow="hidden"
      >
        <Box px="5" py="4" borderBottom="1px solid" borderColor="gray.700">
          <Heading size="sm" color="white">
            Moeda preferida
          </Heading>
          <Text fontSize="xs" color="gray.500" mt="1">
            Define em qual moeda os totais do dashboard são exibidos. Conversão
            automática usa a cotação atual de USD/BRL.
          </Text>
        </Box>
        <Box p="5">
          <form onSubmit={handleSave}>
            <Stack gap="4" maxW="md">
              <Box>
                <Text fontSize="sm" color="gray.300" mb="2">
                  Moeda
                </Text>
                <NativeSelectRoot>
                  <NativeSelectField
                    value={selectedCurrency}
                    onChange={(e) =>
                      setSelectedCurrency(e.target.value as Currency)
                    }
                  >
                    <option value="BRL">{CURRENCY_LABEL.BRL}</option>
                    <option value="USD">{CURRENCY_LABEL.USD}</option>
                  </NativeSelectField>
                </NativeSelectRoot>
              </Box>

              <Flex
                p="3"
                bg="gray.900"
                borderRadius="md"
                border="1px solid"
                borderColor="gray.700"
                gap="3"
                align="center"
              >
                <Box
                  w="6px"
                  h="6px"
                  borderRadius="full"
                  bg={dirty ? "brand.400" : "gray.600"}
                />
                <Text fontSize="xs" color="gray.400">
                  {dirty
                    ? `Alterar para ${selectedCurrency}. Clique em "Salvar" pra confirmar.`
                    : `Atualmente: ${selectedCurrency}.`}
                </Text>
              </Flex>

              <HStack>
                <Button
                  type="submit"
                  colorPalette="blue"
                  loading={saving}
                  disabled={!dirty}
                >
                  Salvar
                </Button>
                {dirty && (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => setSelectedCurrency(initial)}
                    disabled={saving}
                  >
                    Cancelar
                  </Button>
                )}
              </HStack>
            </Stack>
          </form>
        </Box>
      </Box>
    </Stack>
  );
}
