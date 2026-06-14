"use client";

import { createContext, useContext, type ReactNode } from "react";
import type { PluginConfig } from "@/lib/api";

// Carries the effective doc-engine plugin config (enabled + config) down to the
// client MDX components (Kroki, Pre/Mermaid). Server-side plugins (math, alerts,
// toc) are gated in mdx-content.tsx by conditionally including their plugins.
const MdxConfigContext = createContext<PluginConfig | null>(null);

export function MdxConfigProvider({ value, children }: { value: PluginConfig; children: ReactNode }) {
  return <MdxConfigContext.Provider value={value}>{children}</MdxConfigContext.Provider>;
}

export function usePluginConfig(): PluginConfig | null {
  return useContext(MdxConfigContext);
}

// Default to enabled when config is absent or the key is unknown — keeps the
// engine fully functional if the config fetch fails.
export function pluginEnabled(cfg: PluginConfig | null, key: string, fallback = true): boolean {
  if (!cfg || !(key in cfg)) return fallback;
  return cfg[key].enabled;
}

export function pluginValue(cfg: PluginConfig | null, key: string, field: string): string | undefined {
  return cfg?.[key]?.config?.[field];
}

// Uploaded fence plugins: maps a fenced-code language → its JSX source, so the
// client `Pre` can route ```lang blocks to a sandboxed plugin.
const UploadedFencesContext = createContext<Record<string, string>>({});

export function UploadedFencesProvider({ value, children }: { value: Record<string, string>; children: ReactNode }) {
  return <UploadedFencesContext.Provider value={value}>{children}</UploadedFencesContext.Provider>;
}

export function useUploadedFence(lang: string): string | undefined {
  return useContext(UploadedFencesContext)[lang];
}
