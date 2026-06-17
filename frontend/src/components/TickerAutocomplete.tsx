"use client";

import { useEffect, useRef, useState } from "react";
import { Badge, Box, Flex, Input, Spinner, Text } from "@chakra-ui/react";
import { api } from "@/lib/api";
import type { Asset, AssetType, ApiResponse } from "@/types";

interface Props {
  value: string;
  onChange: (ticker: string, asset?: Asset) => void;
  assetType?: AssetType;
  placeholder?: string;
}

const ASSET_LABEL: Record<string, string> = {
  stock: "Ações",
  crypto: "Cripto",
  etf: "ETFs",
  index: "Índices",
};

const BADGE_COLOR: Record<string, string> = {
  stock: "blue",
  crypto: "purple",
  etf: "green",
  index: "orange",
};

export function TickerAutocomplete({
  value,
  onChange,
  assetType,
  placeholder,
}: Props) {
  const [results, setResults] = useState<Asset[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reqIdRef = useRef(0);

  // close on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // debounced search
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    const q = value.trim();
    if (q.length < 1) {
      setResults([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    debounceRef.current = setTimeout(async () => {
      const myReq = ++reqIdRef.current;
      try {
        const params = new URLSearchParams({ q, limit: "10" });
        if (assetType) params.set("type", assetType);
        const res = await api.get<ApiResponse<Asset[]>>(
          `/assets/search?${params.toString()}`,
        );
        if (myReq !== reqIdRef.current) return;
        setResults(Array.isArray(res.data) ? res.data : []);
        setHighlight(0);
      } catch {
        if (myReq !== reqIdRef.current) return;
        // backend down or 5xx: silently degrade
        setResults([]);
      } finally {
        if (myReq === reqIdRef.current) setLoading(false);
      }
    }, 250);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [value, assetType]);

  const select = (a: Asset) => {
    onChange(a.ticker, a);
    setOpen(false);
  };

  const handleKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!open || results.length === 0) {
      if (e.key === "Escape") setOpen(false);
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlight((h) => (h + 1) % results.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlight((h) => (h - 1 + results.length) % results.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      const pick = results[highlight];
      if (pick) select(pick);
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  };

  const showDropdown =
    open && value.trim().length >= 1 && (loading || results.length > 0 || !loading);

  return (
    <Box ref={wrapperRef} position="relative" w="100%">
      <Box position="relative">
        <Input
          value={value}
          onChange={(e) => {
            onChange(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={handleKey}
          placeholder={placeholder}
          autoComplete="off"
          pr={loading ? "10" : undefined}
        />
        {loading && (
          <Box
            position="absolute"
            right="3"
            top="50%"
            transform="translateY(-50%)"
            pointerEvents="none"
          >
            <Spinner size="sm" color="gray.400" />
          </Box>
        )}
      </Box>

      {showDropdown && (
        <Box
          position="absolute"
          top="calc(100% + 4px)"
          left="0"
          right="0"
          zIndex={20}
          bg="gray.800"
          border="1px solid"
          borderColor="gray.700"
          borderRadius="md"
          boxShadow="lg"
          maxH="280px"
          overflowY="auto"
        >
          {results.length === 0 && !loading && (
            <Box px="3" py="3">
              <Text fontSize="sm" color="gray.500">
                Nenhum ativo encontrado
              </Text>
            </Box>
          )}
          {results.map((a, idx) => (
            <Flex
              key={a.id}
              align="center"
              justify="space-between"
              px="3"
              py="2"
              cursor="pointer"
              bg={idx === highlight ? "gray.700" : "transparent"}
              _hover={{ bg: "gray.700" }}
              onMouseEnter={() => setHighlight(idx)}
              onMouseDown={(e) => {
                // mousedown so we beat the outside-click blur
                e.preventDefault();
                select(a);
              }}
            >
              <Text fontWeight="bold" color="white" fontSize="sm">
                {a.ticker}
              </Text>
              <Flex align="center" gap="2" minW={0}>
                <Text
                  fontSize="xs"
                  color="gray.400"
                  maxW="180px"
                  truncate
                  textAlign="right"
                >
                  {a.name}
                </Text>
                <Badge
                  size="sm"
                  variant="subtle"
                  colorPalette={BADGE_COLOR[a.asset_type] ?? "gray"}
                >
                  {ASSET_LABEL[a.asset_type] ?? a.asset_type}
                </Badge>
              </Flex>
            </Flex>
          ))}
        </Box>
      )}
    </Box>
  );
}
