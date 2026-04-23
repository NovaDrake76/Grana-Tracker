"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

// root route just forwards to the portfolios page.
export default function Home() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/dashboard/portfolios");
  }, [router]);

  return null;
}
