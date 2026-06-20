"use client";

import { createContext, useContext, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  LOCALE_COOKIE,
  LOCALE_COOKIE_MAX_AGE,
  locales,
  localeLabels,
  translate,
  type Locale,
  type TranslateFn
} from "./messages";

export { locales, type Locale } from "./messages";

type I18nValue = {
  locale: Locale;
  locales: readonly Locale[];
  localeLabel: (locale: Locale) => string;
  setLocale: (locale: Locale) => void;
  t: TranslateFn;
};

const I18nContext = createContext<I18nValue | null>(null);

export function I18nProvider({ initialLocale, children }: { initialLocale: Locale; children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(initialLocale);
  const router = useRouter();

  const value = useMemo<I18nValue>(() => {
    const setLocale = (next: Locale) => {
      if (next === locale) return;
      setLocaleState(next);
      document.documentElement.lang = next;
      // Persist to a cookie so SSR picks it up, then refresh so server
      // components re-render in the new language too.
      document.cookie = `${LOCALE_COOKIE}=${next}; path=/; max-age=${LOCALE_COOKIE_MAX_AGE}; samesite=lax`;
      router.refresh();
    };
    return {
      locale,
      locales,
      localeLabel: (l) => localeLabels[l],
      setLocale,
      t: (key, vars) => translate(locale, key, vars)
    };
  }, [locale, router]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used inside I18nProvider");
  }
  return ctx;
}
