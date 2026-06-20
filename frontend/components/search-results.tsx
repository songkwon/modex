"use client";

import { BookText, FileText, Layers, Sparkles, CornerDownLeft } from "lucide-react";
import { highlight } from "@/lib/highlight";
import type { useDocSearch } from "@/lib/use-doc-search";
import { useI18n } from "@/lib/i18n";

function entryIcon(type: string) {
  if (type === "vuepress" || type === "fumadocs") return Layers;
  if (type === "static") return FileText;
  return BookText;
}

type Search = ReturnType<typeof useDocSearch>;

// SearchResults renders the Ask-AI row + result list + answer panel for both the
// inline home search and the command palette. The first row (Ask AI) is index 0.
export function SearchResults({ s }: { s: Search }) {
  const { t } = useI18n();
  const { query, results, loading, active, setActive, runAsk, go, answer, setAnswer } = s;

  return (
    <>
      {query.trim() && !answer ? (
        <div className="ds-panel">
          <button
            className={`ds-ask-row ${active === 0 ? "active" : ""}`}
            onMouseEnter={() => setActive(0)}
            onClick={runAsk}
          >
            <Sparkles size={16} className="ds-ask-icon" />
            <span>{t("legacy.09ee788d261e")}<strong>{query}</strong></span>
            <CornerDownLeft size={14} className="muted" />
          </button>
          <div className="ds-results">
            {results.map((r, i) => {
              const Icon = entryIcon(r.entry_type);
              return (
                <button
                  className={`ds-item ${active === i + 1 ? "active" : ""}`}
                  key={r.doc_id}
                  onMouseEnter={() => setActive(i + 1)}
                  onClick={() => go(r.path)}
                >
                  <span className="ds-item-icon"><Icon size={18} /></span>
                  <span className="ds-item-body">
                    <span className="ds-crumb">
                      <span>{r.breadcrumb}</span>
                      {r.docs_version ? <span className="ds-version">{r.docs_version}</span> : null}
                    </span>
                    <span className="ds-title">{highlight(r.title, r.match_terms)}</span>
                    <span className="ds-snippet">{highlight(r.snippet, r.match_terms)}</span>
                  </span>
                </button>
              );
            })}
            {results.length === 0 && !loading ? (
              <div className="ds-empty muted">{t("legacy.5c0318141e37")}</div>
            ) : null}
          </div>
          <div className="ds-hint muted">{t("legacy.1616c5c60e66")}</div>
        </div>
      ) : null}

      {answer ? (
        <div className="ds-answer panel">
          <div className="ds-answer-head">
            <Sparkles size={16} className="ds-ask-icon" />
            <strong>{t("legacy.3f008047e32d")}</strong>
            <span className="tag">{answer.provider}</span>
            <button className="button ds-answer-close" onClick={() => setAnswer(null)}>{t("legacy.3fd47edce45b")}</button>
          </div>
          <div className="ds-answer-body">{answer.answer}</div>
          {(answer.sources?.length ?? 0) > 0 ? (
            <div className="ds-sources">
              <div className="muted ds-sources-label">{t("legacy.e4ec20e4843e")}</div>
              {(answer.sources || []).map((src) => (
                <button className="ds-source" key={src.doc_id} onClick={() => go(src.path)}>
                  <span className="ds-crumb">{src.breadcrumb}</span>
                  <span className="ds-title">{src.title}</span>
                </button>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </>
  );
}
