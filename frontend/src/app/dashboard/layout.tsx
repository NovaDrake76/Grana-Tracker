"use client";

import { useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import {
  Box,
  Flex,
  VStack,
  Text,
  Button,
  Heading,
  Spinner,
  Center,
} from "@chakra-ui/react";
import NextLink from "next/link";
import { useAuth } from "@/context/AuthContext";

const navItems = [
  { label: "Dashboard", href: "/dashboard" },
  { label: "Portfolios", href: "/dashboard/portfolios" },
  { label: "Add Investment", href: "/dashboard/investments/new" },
  { label: "Settings", href: "/dashboard/settings" },
];

// wraps every /dashboard/* page with the sidebar and redirects to /login if not authed.
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isAuthenticated, isLoading, logout, user } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isAuthenticated, isLoading, router]);

  if (isLoading) {
    return (
      <Center h="100vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  if (!isAuthenticated) return null;

  return (
    <Flex h="100vh">
      <Box
        w="250px"
        bg="gray.800"
        p="4"
        display="flex"
        flexDirection="column"
        borderRight="1px solid"
        borderColor="gray.700"
      >
        <Heading size="md" color="brand.500" mb="8" px="2">
          Grana Tracker
        </Heading>

        <VStack gap="1" align="stretch" flex="1">
          {navItems.map((item) => (
            <NextLink key={item.label} href={item.href}>
              <Button
                variant="ghost"
                justifyContent="flex-start"
                fontWeight={pathname === item.href ? "bold" : "normal"}
                bg={pathname === item.href ? "gray.700" : "transparent"}
                _hover={{ bg: "gray.700" }}
                size="sm"
                w="100%"
              >
                {item.label}
              </Button>
            </NextLink>
          ))}
        </VStack>

        <Box borderTop="1px solid" borderColor="gray.700" pt="4">
          <Text fontSize="sm" color="gray.400" mb="2" px="2">
            {user?.name}
          </Text>
          <Button
            variant="ghost"
            size="sm"
            w="100%"
            justifyContent="flex-start"
            color="red.400"
            _hover={{ bg: "gray.700" }}
            onClick={logout}
          >
            Logout
          </Button>
        </Box>
      </Box>

      <Box flex="1" p="8" overflowY="auto">
        {children}
      </Box>
    </Flex>
  );
}
