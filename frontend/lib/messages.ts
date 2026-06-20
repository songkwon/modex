// Locale-agnostic message catalog and translation helpers shared by the
// client provider (lib/i18n.tsx) and the server helper (lib/i18n-server.ts).
// Free of "use client"/server-only imports so both runtimes share one source.
import zhCN from "@/messages/zh-CN.json";
import enUS from "@/messages/en-US.json";

export const locales = ["zh-CN", "en-US"] as const;
export type Locale = (typeof locales)[number];
type Messages = Record<string, string>;

const messages: Record<Locale, Messages> = {
  "zh-CN": zhCN,
  "en-US": enUS
};

export const localeLabels: Record<Locale, string> = {
  "zh-CN": "中文",
  "en-US": "English"
};

// Cookie (not localStorage) so the server can read the preference during SSR
// and render server components in the right language.
export const LOCALE_COOKIE = "modex_locale";
export const LOCALE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

export function normalizeLocale(value: string | null | undefined): Locale {
  if (!value) return "zh-CN";
  const exact = locales.find((locale) => locale === value);
  if (exact) return exact;
  if (value.toLowerCase().startsWith("en")) return "en-US";
  return "zh-CN";
}

function interpolate(template: string, vars?: Record<string, string | number>) {
  if (!vars) return template;
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) => String(vars[key] ?? ""));
}

export function translate(locale: Locale, key: string, vars?: Record<string, string | number>) {
  const dict = messages[locale] || messages["zh-CN"];
  return interpolate(dict[key] || messages["zh-CN"][key] || key, vars);
}

export type TranslateFn = (key: string, vars?: Record<string, string | number>) => string;
