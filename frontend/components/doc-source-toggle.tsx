"use client";

import { useState } from "react";
import { FileCode, Copy, Check, Eye } from "lucide-react";

type ViewMode = "rendered" | "source";

export function DocSourceToggle({
  source,
  children,
}: {
  source: string;
  children: React.ReactNode;
}) {
  const [mode, setMode] = useState<ViewMode>("rendered");
  const [copied, setCopied] = useState(false);

  async function copySource() {
    try {
      await navigator.clipboard.writeText(source);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      alert("复制失败，请手动选中复制");
    }
  }

  return (
    <>
      <div className="doc-view-toggle">
        <button
          className={`doc-view-toggle__btn${mode === "rendered" ? " active" : ""}`}
          onClick={() => setMode("rendered")}
          title="渲染视图"
        >
          <Eye size={14} /> 查看效果
        </button>
        <button
          className={`doc-view-toggle__btn${mode === "source" ? " active" : ""}`}
          onClick={() => setMode("source")}
          title="原始 Markdown"
        >
          <FileCode size={14} /> 查看原始文档
        </button>
      </div>

      {mode === "rendered" ? (
        children
      ) : (
        <div className="doc-source">
          <div className="doc-source__head">
            <span className="doc-source__label">原始 Markdown</span>
            <button className="doc-source__copy" onClick={copySource}>
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? "已复制" : "复制"}
            </button>
          </div>
          <pre className="doc-source__pre">{source}</pre>
        </div>
      )}
    </>
  );
}
