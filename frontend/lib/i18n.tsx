"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import zhCN from "@/messages/zh-CN.json";
import enUS from "@/messages/en-US.json";

export const locales = ["zh-CN", "en-US"] as const;
export type Locale = (typeof locales)[number];
type Messages = Record<string, string>;

const messages: Record<Locale, Messages> = {
  "zh-CN": zhCN,
  "en-US": enUS
};

const localeLabels: Record<Locale, string> = {
  "zh-CN": "中文",
  "en-US": "English"
};

type I18nValue = {
  locale: Locale;
  locales: readonly Locale[];
  localeLabel: (locale: Locale) => string;
  setLocale: (locale: Locale) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
};

const I18nContext = createContext<I18nValue | null>(null);

function normalizeLocale(value: string | null | undefined): Locale {
  if (!value) return "zh-CN";
  const exact = locales.find((locale) => locale === value);
  if (exact) return exact;
  const lower = value.toLowerCase();
  if (lower.startsWith("en")) return "en-US";
  return "zh-CN";
}

function interpolate(template: string, vars?: Record<string, string | number>) {
  if (!vars) return template;
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) => String(vars[key] ?? ""));
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("zh-CN");

  useEffect(() => {
    const stored = window.localStorage.getItem("modex_locale");
    setLocaleState(normalizeLocale(stored || window.navigator.language));
  }, []);

  const value = useMemo<I18nValue>(() => {
    const setLocale = (next: Locale) => {
      setLocaleState(next);
      window.localStorage.setItem("modex_locale", next);
      document.documentElement.lang = next;
    };
    const t = (key: string, vars?: Record<string, string | number>) => {
      const dict = messages[locale] || messages["zh-CN"];
      return interpolate(dict[key] || messages["zh-CN"][key] || key, vars);
    };
    return {
      locale,
      locales,
      localeLabel: (l) => localeLabels[l],
      setLocale,
      t
    };
  }, [locale]);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used inside I18nProvider");
  }
  return ctx;
}
