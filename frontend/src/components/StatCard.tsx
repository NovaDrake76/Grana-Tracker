"use client";

import { Box, Flex, Heading, Text } from "@chakra-ui/react";
import type { ReactNode } from "react";

interface StatCardProps {
  label: string;
  value: string | number;
  helper?: string;
  icon?: ReactNode;
  accent?: "brand" | "gain" | "loss" | "purple" | "gray";
}

const accentMap = {
  brand: { iconBg: "#0c4a6e", iconColor: "#7dd3fc" },
  gain: { iconBg: "#14532d", iconColor: "#86efac" },
  loss: { iconBg: "#7f1d1d", iconColor: "#fca5a5" },
  purple: { iconBg: "#581c87", iconColor: "#d8b4fe" },
  gray: { iconBg: "#374151", iconColor: "#cbd5e1" },
} as const;

export function StatCard({
  label,
  value,
  helper,
  icon,
  accent = "brand",
}: StatCardProps) {
  const tone = accentMap[accent];

  return (
    <Box
      bg="gray.800"
      border="1px solid"
      borderColor="gray.700"
      borderRadius="md"
      p="5"
    >
      <Flex align="center" justify="space-between" mb="3">
        <Text
          fontSize="xs"
          color="gray.400"
          textTransform="uppercase"
          letterSpacing="0.05em"
        >
          {label}
        </Text>
        {icon && (
          <Flex
            w="32px"
            h="32px"
            align="center"
            justify="center"
            borderRadius="md"
            style={{ background: tone.iconBg, color: tone.iconColor }}
          >
            {icon}
          </Flex>
        )}
      </Flex>
      <Heading size="lg" color="white" lineHeight="1.2">
        {value}
      </Heading>
      {helper && (
        <Text fontSize="xs" color="gray.500" mt="1">
          {helper}
        </Text>
      )}
    </Box>
  );
}
