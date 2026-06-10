"use client";

import { Search } from "lucide-react";
import { useSearch } from "@/components/search-provider";

// TopbarSearchButton opens the global command palette (no longer navigates to a
// dedicated /search page). It also shows the keyboard shortcut hint.
export function TopbarSearchButton() {
  const { openSearch, scope } = useSearch();
  const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform);
  return (
    <button className="button topbar-search" onClick={openSearch} aria-label="搜索文档" title="搜索文档 (⌘K)">
      <Search size={16} />
      <span className="topbar-search-label">{scope.label ? `搜索 ${scope.label}` : "搜索文档"}</span>
      <kbd className="topbar-search-kbd">{isMac ? "⌘K" : "Ctrl K"}</kbd>
    </button>
  );
}
