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
  brand: {
    iconBg: "rgba(14, 165, 233, 0.18)",
    iconColor: "#7dd3fc",
    glow: "rgba(14, 165, 233, 0.25)",
    gradient: "linear-gradient(135deg, rgba(14, 165, 233, 0.18), rgba(14, 165, 233, 0))",
  },
  gain: {
    iconBg: "rgba(34, 197, 94, 0.18)",
    iconColor: "#86efac",
    glow: "rgba(34, 197, 94, 0.25)",
    gradient: "linear-gradient(135deg, rgba(34, 197, 94, 0.18), rgba(34, 197, 94, 0))",
  },
  loss: {
    iconBg: "rgba(239, 68, 68, 0.18)",
    iconColor: "#fca5a5",
    glow: "rgba(239, 68, 68, 0.25)",
    gradient: "linear-gradient(135deg, rgba(239, 68, 68, 0.18), rgba(239, 68, 68, 0))",
  },
  purple: {
    iconBg: "rgba(168, 85, 247, 0.18)",
    iconColor: "#d8b4fe",
    glow: "rgba(168, 85, 247, 0.25)",
    gradient: "linear-gradient(135deg, rgba(168, 85, 247, 0.18), rgba(168, 85, 247, 0))",
  },
  gray: {
    iconBg: "rgba(148, 163, 184, 0.15)",
    iconColor: "#cbd5e1",
    glow: "rgba(148, 163, 184, 0.2)",
    gradient: "linear-gradient(135deg, rgba(148, 163, 184, 0.12), rgba(148, 163, 184, 0))",
  },
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
      className="lift"
      position="relative"
      overflow="hidden"
      borderRadius="xl"
      p="5"
      border="1px solid rgba(148, 163, 184, 0.12)"
      style={{
        background: `${tone.gradient}, rgba(15, 23, 42, 0.6)`,
        backdropFilter: "blur(12px)",
        WebkitBackdropFilter: "blur(12px)",
        boxShadow: `0 1px 0 0 rgba(255,255,255,0.04) inset, 0 20px 40px -20px ${tone.glow}`,
      }}
    >
      <Box
        position="absolute"
        top="-30%"
        right="-20%"
        w="60%"
        h="120%"
        opacity="0.4"
        filter="blur(40px)"
        pointerEvents="none"
        style={{ background: tone.iconBg }}
      />
      <Flex align="center" justify="space-between" mb="3" position="relative">
        <Text
          fontSize="xs"
          color="gray.400"
          textTransform="uppercase"
          letterSpacing="0.08em"
          fontWeight="medium"
        >
          {label}
        </Text>
        {icon && (
          <Flex
            w="36px"
            h="36px"
            align="center"
            justify="center"
            borderRadius="lg"
            style={{
              background: tone.iconBg,
              color: tone.iconColor,
              border: `1px solid ${tone.iconBg}`,
            }}
          >
            {icon}
          </Flex>
        )}
      </Flex>
      <Heading
        size="xl"
        color="white"
        lineHeight="1.1"
        position="relative"
      >
        {value}
      </Heading>
      {helper && (
        <Text fontSize="xs" color="gray.500" mt="2" position="relative">
          {helper}
        </Text>
      )}
    </Box>
  );
}
