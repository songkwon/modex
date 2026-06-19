"use client";

import { useEffect, useRef } from "react";
import { Search, Sparkles, Loader2 } from "lucide-react";
import { SearchResults } from "@/components/search-results";
import { useDocSearch, type SearchScope } from "@/lib/use-doc-search";
import { useI18n } from "@/lib/i18n";

export function DocSearch({
  onAiActiveChange,
  moduleKey,
  categoryId,
  placeholder,
  autoFocus
}: {
  onAiActiveChange?: (active: boolean) => void;
  moduleKey?: string;
  categoryId?: string;
  placeholder?: string;
  autoFocus?: boolean;
}) {
  const { t } = useI18n();
  const scope: SearchScope = { moduleKey, categoryId };
  const s = useDocSearch(scope);
  const boxRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (autoFocus) inputRef.current?.focus();
  }, [autoFocus]);

  // Signal focus mode (hide the rest of the page) while the AI conversation is active.
  useEffect(() => {
    onAiActiveChange?.(s.asking || s.answer !== null);
  }, [s.asking, s.answer, onAiActiveChange]);

  return (
    <div className="doc-search" ref={boxRef}>
      <div className="ds-bar">
        <Search size={20} className="ds-bar-icon" />
        <input
          ref={inputRef}
          value={s.query}
          onChange={(e) => s.setQuery(e.target.value)}
          onKeyDown={s.onKeyDown}
          placeholder={placeholder || t("search.placeholder")}
          aria-label={t("search.aria")}
        />
        {s.loading ? <Loader2 size={18} className="ds-spin muted" /> : null}
        <button className="ds-ask-btn" onClick={s.runAsk} disabled={!s.query.trim() || s.asking}>
          {s.asking ? <Loader2 size={16} className="ds-spin" /> : <Sparkles size={16} />}
          {t("search.askAI")}
        </button>
      </div>
      <SearchResults s={s} />
    </div>
  );
}
