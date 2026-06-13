"use client";

import { useEffect, useState } from "react";
import { api } from "./api";

export type Paged<T> = { items: T[]; total: number; page: number; limit: number };

/**
 * Server-side pagination hook for admin list endpoints. Sends
 * ?page=&limit=&keyword= and expects a { items, total, page, limit } envelope.
 * Resets to page 1 whenever the keyword changes; refetches on page/keyword/reload.
 */
export function usePaged<T>(path: string, pageSize: number, keyword: string) {
  const [items, setItems] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);

  // A keyword change should always bring the user back to the first page.
  useEffect(() => {
    setPage(1);
  }, [keyword]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const sep = path.includes("?") ? "&" : "?";
    const url = `${path}${sep}page=${page}&limit=${pageSize}${keyword ? `&keyword=${encodeURIComponent(keyword)}` : ""}`;
    api<Paged<T>>(url)
      .then((r) => {
        if (cancelled) return;
        setItems(r.items || []);
        setTotal(r.total || 0);
        setError("");
      })
      .catch((e) => {
        if (!cancelled) setError(String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [path, pageSize, page, keyword, tick]);

  return {
    items,
    total,
    page,
    setPage,
    error,
    loading,
    reload: () => setTick((t) => t + 1),
    setItems
  };
}
