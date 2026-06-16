"use client";

import { ChakraProvider, Theme, Box } from "@chakra-ui/react";
import {
  Toaster,
  ToastRoot,
  ToastTitle,
  ToastDescription,
  ToastCloseTrigger,
} from "@chakra-ui/react";
import { AuthProvider } from "@/context/AuthContext";
import { system } from "@/lib/theme";
import { toaster } from "@/lib/toaster";

// borderLeftColor por tipo de toast — mantém a faixa colorida do lado
// esquerdo (verde / vermelho / azul) sem deixar o toast inteiro vermelho.
const accentByType: Record<string, string> = {
  error: "#ef4444",
  success: "#22c55e",
  info: "#0ea5e9",
  warning: "#f59e0b",
};

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ChakraProvider value={system}>
      <Theme appearance="dark">
        <AuthProvider>{children}</AuthProvider>
        <Toaster toaster={toaster}>
          {(toast) => (
            <ToastRoot
              display="flex"
              alignItems="flex-start"
              gap="3"
              minW="320px"
              maxW="420px"
              p="4"
              bg="gray.800"
              color="white"
              borderRadius="md"
              borderLeft="4px solid"
              borderLeftColor={accentByType[toast.type ?? "info"] ?? "#0ea5e9"}
              boxShadow="lg"
            >
              <Box flex="1" minW="0">
                <ToastTitle fontWeight="semibold" fontSize="sm">
                  {toast.title}
                </ToastTitle>
                {toast.description && (
                  <ToastDescription fontSize="sm" color="gray.300" mt="1">
                    {toast.description}
                  </ToastDescription>
                )}
              </Box>
              <ToastCloseTrigger />
            </ToastRoot>
          )}
        </Toaster>
      </Theme>
    </ChakraProvider>
  );
}
