"use client";

import { useEffect, useRef } from "react";
import { Search, Loader2, Sparkles } from "lucide-react";
import { SearchResults } from "@/components/search-results";
import { useDocSearch } from "@/lib/use-doc-search";

export type PaletteScope = { moduleKey?: string; categoryId?: string; label?: string };

export function SearchPalette({ open, onClose, scope }: { open: boolean; onClose: () => void; scope: PaletteScope }) {
  const s = useDocSearch({ moduleKey: scope.moduleKey, categoryId: scope.categoryId }, onClose);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      const t = setTimeout(() => inputRef.current?.focus(), 0);
      return () => clearTimeout(t);
    }
    s.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="palette-overlay" onMouseDown={onClose}>
      <div className="palette" onMouseDown={(e) => e.stopPropagation()}>
        <div className="ds-bar palette-bar">
          <Search size={20} className="ds-bar-icon" />
          <input
            ref={inputRef}
            value={s.query}
            onChange={(e) => s.setQuery(e.target.value)}
            onKeyDown={s.onKeyDown}
            placeholder={scope.label ? `在「${scope.label}」内搜索，或向 AI 提问…` : "搜索全部文档，或向 AI 提问…"}
            aria-label="搜索文档或提问"
          />
          {s.loading ? <Loader2 size={18} className="ds-spin muted" /> : null}
          <button className="ds-ask-btn" onClick={s.runAsk} disabled={!s.query.trim() || s.asking}>
            {s.asking ? <Loader2 size={16} className="ds-spin" /> : <Sparkles size={16} />}
            询问 AI
          </button>
        </div>
        <SearchResults s={s} />
      </div>
    </div>
  );
}
