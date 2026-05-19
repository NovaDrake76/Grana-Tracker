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
      <Center h="100vh" bg="gray.900">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  if (!isAuthenticated) return null;

  return (
    <Flex h="100vh" bg="gray.900">
      <Box
        as="aside"
        w="240px"
        p="4"
        display="flex"
        flexDirection="column"
        bg="gray.800"
        borderRight="1px solid"
        borderColor="gray.700"
      >
        <Flex align="center" gap="3" mb="6" px="1">
          <Flex
            w="36px"
            h="36px"
            align="center"
            justify="center"
            bg="brand.600"
            color="white"
            borderRadius="md"
            fontWeight="bold"
            fontSize="lg"
          >
            G
          </Flex>
          <Box>
            <Heading size="sm" color="white" lineHeight="1.1">
              Grana Tracker
            </Heading>
            <Text fontSize="xs" color="gray.400">
              Investimentos
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
                py="2"
                borderRadius="md"
                cursor={item.disabled ? "not-allowed" : "pointer"}
                opacity={item.disabled ? 0.45 : 1}
                bg={active ? "gray.700" : "transparent"}
                color={active ? "white" : "gray.300"}
                fontWeight={active ? "semibold" : "normal"}
                fontSize="sm"
                _hover={
                  item.disabled
                    ? undefined
                    : active
                      ? undefined
                      : { bg: "gray.700", color: "white" }
                }
              >
                <Box color={active ? "brand.400" : "gray.400"}>{item.icon}</Box>
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

        <Box borderTop="1px solid" borderColor="gray.700" pt="3" mt="3">
          <Flex align="center" gap="3" px="1" mb="3">
            <Flex
              w="32px"
              h="32px"
              align="center"
              justify="center"
              bg="gray.700"
              color="white"
              borderRadius="full"
              fontSize="xs"
              fontWeight="bold"
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
            _hover={{ bg: "gray.700", color: "loss" }}
            onClick={logout}
          >
            <LogoutIcon size={16} />
            <Text ml="2">Logout</Text>
          </Button>
        </Box>
      </Box>

      <Box flex="1" overflowY="auto" bg="gray.900">
        <Box maxW="1200px" mx="auto" p={{ base: "5", md: "8" }}>
          {children}
        </Box>
      </Box>
    </Flex>
  );
}
