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
  Badge,
} from "@chakra-ui/react";
import NextLink from "next/link";
import { useAuth } from "@/context/AuthContext";
import {
  DashboardIcon,
  PortfolioIcon,
  SettingsIcon,
  LogoutIcon,
} from "@/components/Icons";
import type { ReactNode } from "react";

const navItems: {
  label: string;
  href: string;
  icon: ReactNode;
  disabled?: boolean;
}[] = [
  { label: "Dashboard", href: "/dashboard", icon: <DashboardIcon /> },
  { label: "Portfolios", href: "/dashboard/portfolios", icon: <PortfolioIcon /> },
  {
    label: "Settings",
    href: "/dashboard/settings",
    icon: <SettingsIcon />,
    disabled: true,
  },
];

function initials(name?: string) {
  if (!name) return "?";
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

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
      <Center h="100vh" className="app-backdrop">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  if (!isAuthenticated) return null;

  return (
    <Flex h="100vh" className="app-backdrop">
      <Box
        as="aside"
        w="260px"
        p="5"
        display="flex"
        flexDirection="column"
        position="relative"
        zIndex="2"
        bg="rgba(2, 6, 23, 0.6)"
        backdropFilter="blur(20px)"
        borderRight="1px solid"
        borderColor="rgba(148, 163, 184, 0.1)"
      >
        <Flex align="center" gap="3" mb="8" px="1">
          <Flex
            w="40px"
            h="40px"
            align="center"
            justify="center"
            borderRadius="lg"
            fontWeight="bold"
            fontSize="xl"
            color="gray.900"
            style={{
              background: "linear-gradient(135deg, #38bdf8, #0ea5e9)",
              boxShadow:
                "0 0 24px -4px rgba(14, 165, 233, 0.6), inset 0 1px 0 0 rgba(255,255,255,0.3)",
            }}
          >
            G
          </Flex>
          <Box>
            <Heading size="sm" color="white" lineHeight="1.1">
              Grana Tracker
            </Heading>
            <Text fontSize="xs" color="gray.500">
              Investimentos unificados
            </Text>
          </Box>
        </Flex>

        <VStack gap="1" align="stretch" flex="1">
          {navItems.map((item) => {
            const active = pathname === item.href;
            const inner = (
              <Flex
                align="center"
                gap="3"
                px="3"
                py="2.5"
                borderRadius="md"
                cursor={item.disabled ? "not-allowed" : "pointer"}
                opacity={item.disabled ? 0.45 : 1}
                color={active ? "white" : "gray.300"}
                fontWeight={active ? "semibold" : "medium"}
                fontSize="sm"
                position="relative"
                className={active ? "nav-active" : undefined}
                transition="background 0.15s, color 0.15s"
                _hover={
                  item.disabled
                    ? undefined
                    : active
                      ? undefined
                      : { bg: "rgba(148, 163, 184, 0.08)", color: "white" }
                }
              >
                <Box color={active ? "brand.300" : "gray.400"}>{item.icon}</Box>
                <Text>{item.label}</Text>
                {item.disabled && (
                  <Badge
                    ml="auto"
                    size="sm"
                    variant="subtle"
                    colorPalette="gray"
                    fontSize="2xs"
                  >
                    Soon
                  </Badge>
                )}
              </Flex>
            );

            if (item.disabled) {
              return <Box key={item.label}>{inner}</Box>;
            }
            return (
              <NextLink key={item.label} href={item.href}>
                {inner}
              </NextLink>
            );
          })}
        </VStack>

        <Box
          borderTop="1px solid"
          borderColor="rgba(148, 163, 184, 0.1)"
          pt="4"
          mt="4"
        >
          <Flex align="center" gap="3" px="1" mb="3">
            <Flex
              w="36px"
              h="36px"
              align="center"
              justify="center"
              color="white"
              borderRadius="full"
              fontSize="sm"
              fontWeight="bold"
              style={{
                background: "linear-gradient(135deg, #0369a1, #075985)",
                boxShadow: "inset 0 1px 0 0 rgba(255,255,255,0.15)",
              }}
            >
              {initials(user?.name)}
            </Flex>
            <Box flex="1" minW="0">
              <Text fontSize="sm" color="white" lineHeight="1.1" truncate>
                {user?.name ?? "Conta"}
              </Text>
              <Text fontSize="xs" color="gray.500" truncate>
                {user?.email}
              </Text>
            </Box>
          </Flex>
          <Button
            variant="ghost"
            size="sm"
            w="100%"
            justifyContent="flex-start"
            color="gray.400"
            _hover={{ bg: "rgba(239, 68, 68, 0.1)", color: "loss" }}
            onClick={logout}
          >
            <LogoutIcon size={16} />
            <Text ml="2">Logout</Text>
          </Button>
        </Box>
      </Box>

      <Box
        flex="1"
        overflowY="auto"
        position="relative"
        zIndex="1"
      >
        <Box
          maxW="1200px"
          mx="auto"
          p={{ base: "5", md: "8" }}
          className="above-backdrop"
        >
          {children}
        </Box>
      </Box>
    </Flex>
  );
}
