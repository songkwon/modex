"use client";

import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { useRouter } from "next/navigation";
import { askAIStream, searchDocs } from "@/lib/api";
import type { AskResponse, SearchResult } from "@/types/modex";

export type SearchScope = { moduleKey?: string; categoryId?: string };

// useDocSearch powers both the inline home search and the command palette.
// Selection model: index 0 is the "Ask AI" option; indexes 1..n are results.
export function useDocSearch(scope: SearchScope = {}, onNavigate?: () => void) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [answer, setAnswer] = useState<AskResponse | null>(null);
  const [asking, setAsking] = useState(false);
  const [active, setActive] = useState(0);
  const debounce = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const searchSeq = useRef(0);

  const { moduleKey, categoryId } = scope;

  useEffect(() => {
    clearTimeout(debounce.current);
    const seq = ++searchSeq.current;
    const q = query.trim();
    if (!q) {
      setResults([]);
      setLoading(false);
      setActive(0);
      setAnswer(null);
      return;
    }
    setLoading(true);
    debounce.current = setTimeout(async () => {
      try {
        const filters: Record<string, unknown> = {};
        if (moduleKey) filters.modules = [moduleKey];
        if (categoryId) filters.category_ids = [categoryId];
        const body: Record<string, unknown> = { query: q, mode: "hybrid", page_size: 8 };
        if (Object.keys(filters).length) body.filters = filters;
        const res = await searchDocs(body);
        if (seq !== searchSeq.current) return;
        setResults(res.results || []);
        setActive(0);
      } catch {
        if (seq !== searchSeq.current) return;
        setResults([]);
      } finally {
        if (seq === searchSeq.current) setLoading(false);
      }
    }, 200);
    return () => clearTimeout(debounce.current);
  }, [query, moduleKey, categoryId]);

  // logSearch records one search-log row for an explicit, user-committed search
  // (Enter / result click / Ask AI). Live as-you-type queries never log, so the
  // search log isn't flooded with one row per keystroke. Fire-and-forget.
  function logSearch(clickedDocId?: string) {
    const q = query.trim();
    if (!q) return;
    const filters: Record<string, unknown> = {};
    if (moduleKey) filters.modules = [moduleKey];
    if (categoryId) filters.category_ids = [categoryId];
    const body: Record<string, unknown> = { query: q, mode: "hybrid", page_size: 1, log: true };
    if (clickedDocId) body.clicked_doc_id = clickedDocId;
    if (Object.keys(filters).length) body.filters = filters;
    searchDocs(body).catch(() => {});
  }

  async function runAsk() {
    if (!query.trim()) return;
    logSearch();
    setAsking(true);
    setAnswer(null);
    setResults([]);
    setActive(0);
    searchSeq.current += 1;
    clearTimeout(debounce.current);
    try {
      const scopeArg = moduleKey || categoryId ? { module_key: moduleKey, category_ids: categoryId ? [categoryId] : undefined } : undefined;
      await askAIStream(query, scopeArg, setAnswer);
    } catch (e) {
      setAnswer({ query, answer: String(e), provider: "error", sources: [] });
    } finally {
      setAsking(false);
    }
  }

  function go(path: string, docId?: string) {
    logSearch(docId);
    onNavigate?.();
    router.push(path);
  }

  // selectActive triggers the currently highlighted item (Ask AI or a result).
  function selectActive() {
    if (active === 0) {
      runAsk();
      return;
    }
    const r = results[active - 1];
    if (r) go(r.path, r.doc_id);
  }

  function onKeyDown(e: KeyboardEvent) {
    const max = results.length; // index 0..results.length
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((i) => Math.min(i + 1, max));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      selectActive();
    }
  }

  function reset() {
    setQuery("");
    setResults([]);
    setAnswer(null);
    setActive(0);
  }

  return {
    query,
    setQuery,
    results,
    loading,
    answer,
    setAnswer,
    asking,
    active,
    setActive,
    runAsk,
    go,
    onKeyDown,
    reset
  };
}
