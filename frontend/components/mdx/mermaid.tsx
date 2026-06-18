"use client";

import { useEffect, useId, useRef, useState } from "react";

function extractText(node: unknown): string {
  if (node == null) return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(extractText).join("");
  if (typeof node === "object" && "props" in (node as any)) {
    return extractText((node as any).props?.children);
  }
  return "";
}

export function Mermaid({ chart, children }: { chart?: string; children?: React.ReactNode }) {
  const code = (chart ?? extractText(children)).trim();
  const ref = useRef<HTMLDivElement>(null);
  const id = useId().replace(/[:]/g, "");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const mermaid = (await import("mermaid")).default;
        const isDark = document.documentElement.classList.contains("dark") || document.documentElement.getAttribute("data-theme") === "dark";
        mermaid.initialize({ startOnLoad: false, theme: isDark ? "dark" : "neutral", securityLevel: "loose" });
        const { svg } = await mermaid.render(`mmd-${id}`, code);
        if (!cancelled && ref.current) ref.current.innerHTML = svg;
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "diagram error");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [code, id]);

  if (error) {
    return <pre className="mdx-mermaid mdx-mermaid--error">{code}</pre>;
  }
  return <div className="mdx-mermaid" ref={ref} />;
}
