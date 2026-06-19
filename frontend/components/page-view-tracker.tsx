"use client";

import { useEffect, useRef } from "react";
import { capture } from "@/lib/analytics";
import { recordRecentDoc } from "@/lib/local-docs";

// Reading statistics are captured by PostHog. When PostHog is not configured,
// capture() is a no-op and no reading data is collected.
export function PageViewTracker({
  docId,
  title,
  moduleKey,
  moduleName,
  docsVersion,
  entryKey,
}: {
  docId: string;
  title?: string;
  moduleKey?: string;
  moduleName?: string;
  docsVersion?: string;
  entryKey?: string;
}) {
  const startedAt = useRef(Date.now());
  const maxScroll = useRef(0);
  const readId = useRef("");

  useEffect(() => {
    startedAt.current = Date.now();
    maxScroll.current = 0;
    readId.current = crypto.randomUUID();
    capture("docs_page_view", { doc_id: docId, read_id: readId.current });
    if (title && moduleKey && docsVersion && entryKey) {
      recordRecentDoc({
        doc_id: docId,
        title,
        module_key: moduleKey,
        module_name: moduleName || moduleKey,
        docs_version: docsVersion,
        entry_key: entryKey,
        href: `/docs/${moduleKey}/${docsVersion}/${entryKey}`,
      });
    }

    const onScroll = () => {
      const scrollable = document.documentElement.scrollHeight - window.innerHeight;
      const depth = scrollable > 0 ? Math.min(1, window.scrollY / scrollable) : 1;
      maxScroll.current = Math.max(maxScroll.current, depth);
    };
    const flush = () => {
      capture("docs_page_read", {
        doc_id: docId,
        read_id: readId.current,
        duration_seconds: Math.max(0, Math.round((Date.now() - startedAt.current) / 1000)),
        scroll_depth: Number(maxScroll.current.toFixed(2)),
      });
    };
    const onVisibilityChange = () => {
      if (document.visibilityState === "hidden") flush();
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      window.removeEventListener("scroll", onScroll);
      document.removeEventListener("visibilitychange", onVisibilityChange);
      flush();
    };
  }, [docId, docsVersion, entryKey, moduleKey, moduleName, title]);

  return null;
}
