"use client";

import { useEffect, useRef, useState } from "react";

// Diagram-as-code rendered via a Kroki server (https://kroki.io). One HTTP API
// covers PlantUML, Graphviz, C4, DITAA, BPMN, Excalidraw, Vega, D2 and more.
// The base URL is configurable so deployments can point at a self-hosted Kroki
// instance and keep diagram source off the public service.
const KROKI_BASE = (process.env.NEXT_PUBLIC_KROKI_URL || "https://kroki.io").replace(/\/+$/, "");

// Fenced-code language → Kroki diagram type. Aliases map to canonical types.
const KROKI_TYPES: Record<string, string> = {
  plantuml: "plantuml",
  puml: "plantuml",
  uml: "plantuml",
  c4: "c4plantuml",
  c4plantuml: "c4plantuml",
  graphviz: "graphviz",
  dot: "graphviz",
  ditaa: "ditaa",
  blockdiag: "blockdiag",
  seqdiag: "seqdiag",
  actdiag: "actdiag",
  nwdiag: "nwdiag",
  packetdiag: "packetdiag",
  rackdiag: "rackdiag",
  bpmn: "bpmn",
  bytefield: "bytefield",
  excalidraw: "excalidraw",
  nomnoml: "nomnoml",
  pikchr: "pikchr",
  structurizr: "structurizr",
  svgbob: "svgbob",
  vega: "vega",
  vegalite: "vegalite",
  "vega-lite": "vegalite",
  wavedrom: "wavedrom",
  wireviz: "wireviz",
  d2: "d2",
  dbml: "dbml",
  erd: "erd"
};

export function isKrokiLang(lang: string): boolean {
  return lang.toLowerCase() in KROKI_TYPES;
}

export function Kroki({ lang, code }: { lang: string; code: string }) {
  const type = KROKI_TYPES[lang.toLowerCase()] || lang.toLowerCase();
  const ref = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);
  const source = code.trim();

  useEffect(() => {
    let cancelled = false;
    setError(null);
    (async () => {
      try {
        const res = await fetch(`${KROKI_BASE}/${type}/svg`, {
          method: "POST",
          headers: { "Content-Type": "text/plain", Accept: "image/svg+xml" },
          body: source
        });
        if (!res.ok) throw new Error(`Kroki ${res.status}: ${(await res.text()).slice(0, 200)}`);
        const svg = await res.text();
        if (!cancelled && ref.current) ref.current.innerHTML = svg;
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "diagram error");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [type, source]);

  if (error) {
    return (
      <div className="mdx-diagram mdx-diagram--error">
        <span className="mdx-diagram__tag">{type} 渲染失败</span>
        <pre>{source}</pre>
      </div>
    );
  }
  return <div className="mdx-diagram" data-kroki={type} ref={ref} />;
}
