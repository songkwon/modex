"use client";

import { Search } from "lucide-react";
import { useSearch } from "@/components/search-provider";
import { useI18n } from "@/lib/i18n";

// TopbarSearchButton opens the global command palette (no longer navigates to a
// dedicated /search page). It also shows the keyboard shortcut hint.
export function TopbarSearchButton() {
  const { t } = useI18n();
  const { openSearch, scope } = useSearch();
  const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform);
  return (
    <button className="button topbar-search" onClick={openSearch} aria-label={t("nav.search")} title={t("nav.searchTitle")}>
      <Search size={16} />
      <span className="topbar-search-label">{scope.label ? t("nav.searchScoped", { scope: scope.label }) : t("nav.search")}</span>
      <kbd className="topbar-search-kbd">{isMac ? "⌘K" : "Ctrl K"}</kbd>
    </button>
  );
}
