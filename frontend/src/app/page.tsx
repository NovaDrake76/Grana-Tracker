"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

// root route forwards to the dashboard home (auth check happens inside the layout).
export default function Home() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/dashboard");
  }, [router]);

  return null;
}
