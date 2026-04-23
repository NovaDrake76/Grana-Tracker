"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

// placeholder until the real dashboard lands; redirects to portfolios for now.
export default function DashboardPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/dashboard/portfolios");
  }, [router]);

  return null;
}
