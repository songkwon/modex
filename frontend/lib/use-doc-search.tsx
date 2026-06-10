"use client";

import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { useRouter } from "next/navigation";
import { askAI, searchDocs } from "@/lib/api";
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
  const debounce = useRef<ReturnType<typeof setTimeout>>();

  const { moduleKey, categoryId } = scope;

  useEffect(() => {
    clearTimeout(debounce.current);
    if (!query.trim()) {
      setResults([]);
      setActive(0);
      return;
    }
    setLoading(true);
    debounce.current = setTimeout(async () => {
      try {
        const filters: Record<string, unknown> = {};
        if (moduleKey) filters.modules = [moduleKey];
        if (categoryId) filters.category_ids = [categoryId];
        const body: Record<string, unknown> = { query, mode: "hybrid", page_size: 8 };
        if (Object.keys(filters).length) body.filters = filters;
        const res = await searchDocs(body);
        setResults(res.results || []);
        setActive(0);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 200);
    return () => clearTimeout(debounce.current);
  }, [query, moduleKey, categoryId]);

  async function runAsk() {
    if (!query.trim()) return;
    setAsking(true);
    setAnswer(null);
    try {
      const scopeArg = moduleKey || categoryId ? { module_key: moduleKey, category_ids: categoryId ? [categoryId] : undefined } : undefined;
      setAnswer(await askAI(query, scopeArg));
    } catch (e) {
      setAnswer({ query, answer: String(e), provider: "error", sources: [] });
    } finally {
      setAsking(false);
    }
  }

  function go(path: string) {
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
    if (r) go(r.path);
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
