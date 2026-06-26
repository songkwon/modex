"use client";

import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { usePluginConfig, pluginValue } from "./mdx-config";
import { useI18n } from "@/lib/i18n";
import { publicKrokiURL } from "@/lib/runtime-config";

// Diagram-as-code rendered via a Kroki server (https://kroki.io). One HTTP API
// covers PlantUML, Graphviz, C4, DITAA, BPMN, Excalidraw, Vega, D2 and more.
// The base URL is configurable (admin plugin config, then env) so deployments
// can point at a self-hosted Kroki instance and keep diagram source private.
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
  const { t } = useI18n();
  const type = KROKI_TYPES[lang.toLowerCase()] || lang.toLowerCase();
  const cfg = usePluginConfig();
  const base = (pluginValue(cfg, "kroki", "base_url") || publicKrokiURL()).replace(/\/+$/, "");
  const ref = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [svg, setSvg] = useState("");
  const [open, setOpen] = useState(false);
  const source = code.trim();

  useEffect(() => {
    let cancelled = false;
    setError(null);
    (async () => {
      try {
        const res = await fetch(`${base}/${type}/svg`, {
          method: "POST",
          headers: { "Content-Type": "text/plain", Accept: "image/svg+xml" },
          body: source
        });
        if (!res.ok) throw new Error(`Kroki ${res.status}: ${(await res.text()).slice(0, 200)}`);
        const svg = await res.text();
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
  }, [type, source, base]);

  if (error) {
    return (
      <div className="mdx-diagram mdx-diagram--error">
        <span className="mdx-diagram__tag">{type} {t("component.mdx.kroki.rendering_failed")}</span>
        <pre>{source}</pre>
      </div>
    );
  }
  return (
    <>
      <button className="mdx-diagram-trigger" type="button" onClick={() => svg && setOpen(true)}>
        <div className="mdx-diagram" data-kroki={type} ref={ref} />
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
