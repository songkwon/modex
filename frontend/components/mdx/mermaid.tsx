"use client";

import { useEffect, useId, useRef, useState } from "react";
import { X } from "lucide-react";

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
  const [svg, setSvg] = useState("");
  const [open, setOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const mermaid = (await import("mermaid")).default;
        const isDark = document.documentElement.classList.contains("dark") || document.documentElement.getAttribute("data-theme") === "dark";
        mermaid.initialize({ startOnLoad: false, theme: isDark ? "dark" : "neutral", securityLevel: "loose" });
        const { svg } = await mermaid.render(`mmd-${id}`, code);
        if (!cancelled) {
          setSvg(svg);
          if (ref.current) ref.current.innerHTML = svg;
        }
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
  return (
    <>
      <button className="mdx-diagram-trigger" type="button" onClick={() => svg && setOpen(true)}>
        <div className="mdx-mermaid" ref={ref} />
      </button>
      {open ? (
        <div className="mdx-image-preview" role="dialog" aria-modal="true" aria-label="Diagram preview" onClick={() => setOpen(false)}>
          <button className="mdx-image-preview__close" type="button" aria-label="Close diagram preview" onClick={() => setOpen(false)}>
            <X size={20} />
          </button>
          <div className="mdx-diagram-preview__svg" dangerouslySetInnerHTML={{ __html: svg }} onClick={(event) => event.stopPropagation()} />
        </div>
      ) : null}
    </>
  );
}
