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
      alert(t("component.docSourceToggle.copy_failed_please_manually_select_and_copy"));
    }
  }

  return (
    <>
      <div className="doc-view-toggle">
        <button
          className={`doc-view-toggle__btn${mode === "rendered" ? " active" : ""}`}
          onClick={() => setMode("rendered")}
          title={t("component.docSourceToggle.render_view")}
        >
          <Eye size={14} /> {t("component.docSourceToggle.view_results")}
        </button>
        <button
          className={`doc-view-toggle__btn${mode === "source" ? " active" : ""}`}
          onClick={() => setMode("source")}
          title={t("component.docSourceToggle.original_markdown")}
        >
          <FileCode size={14} /> {t("component.docSourceToggle.view_original_document")}
        </button>
      </div>

      {mode === "rendered" ? (
        children
      ) : (
        <div className="doc-source">
          <div className="doc-source__head">
            <span className="doc-source__label">{t("component.docSourceToggle.original_markdown")}</span>
            <button className="doc-source__copy" onClick={copySource}>
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? t("component.ui.copyButton.copied") : t("component.ui.copyButton.copy")}
            </button>
          </div>
          <pre className="doc-source__pre">{source}</pre>
        </div>
      )}
    </>
  );
}
