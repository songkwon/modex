"use client";

import { useEffect, useRef } from "react";
import { recordPageView, recordReadProgress } from "@/lib/api";
import { capture, sessionId } from "@/lib/analytics";

// PageViewTracker records a page view on mount and flushes read progress
// (dwell time + scroll depth) on unmount or when the tab is hidden.
export function PageViewTracker({ docId }: { docId: string }) {
  const startedAt = useRef<number>(Date.now());
  const maxScroll = useRef<number>(0);

  useEffect(() => {
    const sid = sessionId();
    startedAt.current = Date.now();

    recordPageView({ doc_id: docId, session_id: sid }).catch(() => {});
    capture("docs_page_view", { doc_id: docId });

    const onScroll = () => {
      const scrollable = document.documentElement.scrollHeight - window.innerHeight;
      const depth = scrollable > 0 ? Math.min(1, window.scrollY / scrollable) : 1;
      if (depth > maxScroll.current) maxScroll.current = depth;
    };
    window.addEventListener("scroll", onScroll, { passive: true });

    const flush = () => {
      const duration = Math.round((Date.now() - startedAt.current) / 1000);
      recordReadProgress({
        doc_id: docId,
        session_id: sid,
        duration_seconds: duration,
        scroll_depth: Number(maxScroll.current.toFixed(2))
      }).catch(() => {});
    };
    const onHidden = () => {
      if (document.visibilityState === "hidden") flush();
    };
    document.addEventListener("visibilitychange", onHidden);

    return () => {
      window.removeEventListener("scroll", onScroll);
      document.removeEventListener("visibilitychange", onHidden);
      flush();
    };
  }, [docId]);

  return null;
}
