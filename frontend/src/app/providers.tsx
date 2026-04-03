"use client";

import { ChakraProvider, Theme } from "@chakra-ui/react";
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

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ChakraProvider value={system}>
      <Theme appearance="dark">
        <AuthProvider>{children}</AuthProvider>
        <Toaster toaster={toaster}>
          {(toast) => (
            <ToastRoot>
              <ToastTitle>{toast.title}</ToastTitle>
              {toast.description && (
                <ToastDescription>{toast.description}</ToastDescription>
              )}
              <ToastCloseTrigger />
            </ToastRoot>
          )}
        </Toaster>
      </Theme>
    </ChakraProvider>
  );
}
