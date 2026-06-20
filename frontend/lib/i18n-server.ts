// Server-side translation: read the locale cookie during SSR so server
// components render in the visitor's language. Client components use the
// useI18n() hook from lib/i18n.tsx instead.
import { cookies } from "next/headers";
import { LOCALE_COOKIE, normalizeLocale, translate, type Locale, type TranslateFn } from "./messages";

export async function getServerLocale(): Promise<Locale> {
  const store = await cookies();
  return normalizeLocale(store.get(LOCALE_COOKIE)?.value);
}

export async function getServerI18n(): Promise<{ locale: Locale; t: TranslateFn }> {
  const locale = await getServerLocale();
  return { locale, t: (key, vars) => translate(locale, key, vars) };
}
