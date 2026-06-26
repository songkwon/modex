"use client";

import { BookText, FileText, Layers, Sparkles, CornerDownLeft } from "lucide-react";
import { highlight } from "@/lib/highlight";
import type { useDocSearch } from "@/lib/use-doc-search";
import { useI18n } from "@/lib/i18n";
import { AiMarkdown, splitAnswerParts } from "@/components/ai-markdown";

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
  const { query, results, loading, asking, active, setActive, runAsk, go, answer, setAnswer } = s;
  const answerParts = answer ? splitAnswerParts(answer) : null;

  return (
    <>
      {query.trim() && !answer && !asking ? (
        <div className="ds-panel">
          <button
            className={`ds-ask-row ${active === 0 ? "active" : ""}`}
            onMouseEnter={() => setActive(0)}
            onClick={runAsk}
          >
            <Sparkles size={16} className="ds-ask-icon" />
            <span>{t("component.searchResults.ask_ai")}<strong>{query}</strong></span>
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
                  onClick={() => go(r.path, r.doc_id)}
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
              <div className="ds-empty muted">{t("component.searchResults.no_matching_documents_press_enter_or_click_ask")}</div>
            ) : null}
          </div>
          <div className="ds-hint muted">{t("component.searchResults.to_select_enter_to_open_esc_to_close")}</div>
        </div>
      ) : null}

      {answer ? (
        <div className="ds-answer panel">
          <div className="ds-answer-head">
            <Sparkles size={16} className="ds-ask-icon" />
            <strong>{t("component.searchResults.ai_response")}</strong>
            <span className="tag">{providerLabel(answer.provider)}</span>
            <button className="button ds-answer-close" onClick={() => setAnswer(null)}>{t("component.searchResults.close")}</button>
          </div>
          <div className="ds-answer-body">
            {answer.warning ? <div className="panel badge-warn" style={{ marginBottom: 10, borderRadius: 10 }}>{answer.warning}</div> : null}
            {answerParts ? <AiMarkdown answer={answerParts.answer} reasoning={answerParts.reasoning} /> : null}
          </div>
          {answer.done && (answer.sources?.length ?? 0) > 0 ? (
            <div className="ds-sources">
              <div className="muted ds-sources-label">{t("component.searchResults.reference_documentation")}</div>
              {(answer.sources || []).map((src) => (
                <button className="ds-source" key={src.doc_id} onClick={() => go(src.path, src.doc_id)}>
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

function providerLabel(provider: string) {
  if (provider === "llm") return "AI 问答";
  if (provider === "extractive") return "本地摘要";
  if (provider === "error") return "错误";
  return provider || "AI 问答";
}
