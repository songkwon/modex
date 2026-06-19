"use client";

import { Languages } from "lucide-react";
import { useI18n, type Locale } from "@/lib/i18n";

export function LocaleSwitcher() {
  const { locale, locales, localeLabel, setLocale, t } = useI18n();

  return (
    <label className="locale-switcher" title={t("nav.language")} aria-label={t("nav.language")}>
      <Languages size={16} />
      <select data-testid="locale-select" aria-label={t("nav.language")} value={locale} onChange={(e) => setLocale(e.target.value as Locale)}>
        {locales.map((item) => (
          <option key={item} value={item}>
            {localeLabel(item)}
          </option>
        ))}
      </select>
    </label>
  );
}
