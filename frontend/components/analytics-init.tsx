"use client";

import { useEffect } from "react";
import { initAnalytics } from "@/lib/analytics";

// AnalyticsInit initializes PostHog once per browser session when
// MODEX_PUBLIC_POSTHOG_KEY is set. This must be a client component because
// posthog-js reads/writes browser storage and cookies.
export function AnalyticsInit() {
  useEffect(() => {
    initAnalytics();
  }, []);
  return null;
}
