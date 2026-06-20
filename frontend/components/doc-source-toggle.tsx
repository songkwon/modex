"use client";

import { useState } from "react";
import { FileCode, Copy, Check, Eye } from "lucide-react";
import { useI18n } from "@/lib/i18n";

type ViewMode = "rendered" | "source";

export function DocSourceToggle({
  source,
  children,
}: {
  source: string;
  children: React.ReactNode;
}) {
  const { t } = useI18n();
  const [mode, setMode] = useState<ViewMode>("rendered");
  const [copied, setCopied] = useState(false);

  async function copySource() {
    try {
      await navigator.clipboard.writeText(source);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      alert(t("legacy.cbc8618279a5"));
    }
  }

  return (
    <>
      <div className="doc-view-toggle">
        <button
          className={`doc-view-toggle__btn${mode === "rendered" ? " active" : ""}`}
          onClick={() => setMode("rendered")}
          title={t("legacy.4390fa32fde0")}
        >
          <Eye size={14} /> {t("legacy.c990c85989ec")}
        </button>
        <button
          className={`doc-view-toggle__btn${mode === "source" ? " active" : ""}`}
          onClick={() => setMode("source")}
          title={t("legacy.7d8081dad59b")}
        >
          <FileCode size={14} /> {t("legacy.a57a3a44d9c3")}
        </button>
      </div>

      {mode === "rendered" ? (
        children
      ) : (
        <div className="doc-source">
          <div className="doc-source__head">
            <span className="doc-source__label">{t("legacy.7d8081dad59b")}</span>
            <button className="doc-source__copy" onClick={copySource}>
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? t("legacy.8f6f8d979c98") : t("legacy.63d90d977348")}
            </button>
          </div>
          <pre className="doc-source__pre">{source}</pre>
        </div>
      )}
    </>
  );
}
