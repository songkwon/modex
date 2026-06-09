"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { BookText, FileText, Layers, Search, Sparkles, Loader2, CornerDownLeft } from "lucide-react";
import { askAI, searchDocs } from "@/lib/api";
import { highlight } from "@/lib/highlight";
import type { AskResponse, SearchResult } from "@/types/modex";

function entryIcon(type: string) {
  if (type === "vuepress" || type === "fumadocs") return Layers;
  if (type === "static") return FileText;
  return BookText;
}

export function DocSearch({
  onAiActiveChange,
  moduleKey,
  placeholder
}: {
  onAiActiveChange?: (active: boolean) => void;
  moduleKey?: string;
  placeholder?: string;
}) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [answer, setAnswer] = useState<AskResponse | null>(null);
  const [asking, setAsking] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  const debounce = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  // Signal focus mode (hide the rest of the page) while the AI conversation is active.
  useEffect(() => {
    onAiActiveChange?.(asking || answer !== null);
  }, [asking, answer, onAiActiveChange]);

  useEffect(() => {
    clearTimeout(debounce.current);
    if (!query.trim()) {
      setResults([]);
      return;
    }
    setLoading(true);
    debounce.current = setTimeout(async () => {
      try {
        const body: Record<string, unknown> = { query, mode: "hybrid", page_size: 8 };
        if (moduleKey) body.filters = { modules: [moduleKey] };
        const res = await searchDocs(body);
        setResults(res.results || []);
        setOpen(true);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 220);
    return () => clearTimeout(debounce.current);
  }, [query, moduleKey]);

  async function runAsk() {
    if (!query.trim()) return;
    setAsking(true);
    setAnswer(null);
    setOpen(false);
    try {
      setAnswer(await askAI(query));
    } catch (e) {
      setAnswer({ query, answer: String(e), provider: "error", sources: [] });
    } finally {
      setAsking(false);
    }
  }

  function go(path: string) {
    setOpen(false);
    router.push(path);
  }

  return (
    <div className="doc-search" ref={boxRef}>
      <div className="ds-bar">
        <Search size={20} className="ds-bar-icon" />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => results.length && setOpen(true)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              if (results[0]) go(results[0].path);
            }
          }}
          placeholder={placeholder || "搜索文档，或向 AI 提问…"}
          aria-label="搜索文档或提问"
        />
        {loading ? <Loader2 size={18} className="ds-spin muted" /> : null}
        <button className="ds-ask-btn" onClick={runAsk} disabled={!query.trim() || asking}>
          {asking ? <Loader2 size={16} className="ds-spin" /> : <Sparkles size={16} />}
          询问 AI
        </button>
      </div>

      {open && (results.length > 0 || query.trim()) ? (
        <div className="ds-panel">
          <button className="ds-ask-row" onClick={runAsk}>
            <Sparkles size={16} className="ds-ask-icon" />
            <span>向 AI 提问：<strong>{query}</strong></span>
            <CornerDownLeft size={14} className="muted" />
          </button>
          <div className="ds-results">
            {results.map((r) => {
              const Icon = entryIcon(r.entry_type);
              return (
                <button className="ds-item" key={r.doc_id} onClick={() => go(r.path)}>
                  <span className="ds-item-icon"><Icon size={18} /></span>
                  <span className="ds-item-body">
                    <span className="ds-crumb">{r.breadcrumb}</span>
                    <span className="ds-title">{highlight(r.title, r.match_terms)}</span>
                    <span className="ds-snippet">{highlight(r.snippet, r.match_terms)}</span>
                  </span>
                </button>
              );
            })}
            {results.length === 0 && !loading ? <div className="ds-empty muted">没有匹配的文档，试试点击「询问 AI」。</div> : null}
          </div>
        </div>
      ) : null}

      {answer ? (
        <div className="ds-answer panel">
          <div className="ds-answer-head">
            <Sparkles size={16} className="ds-ask-icon" />
            <strong>AI 回答</strong>
            <span className="tag">{answer.provider}</span>
            <button className="button ds-answer-close" onClick={() => setAnswer(null)}>关闭</button>
          </div>
          <div className="ds-answer-body">{answer.answer}</div>
          {answer.sources.length > 0 ? (
            <div className="ds-sources">
              <div className="muted ds-sources-label">引用文档</div>
              {answer.sources.map((s) => (
                <button className="ds-source" key={s.doc_id} onClick={() => go(s.path)}>
                  <span className="ds-crumb">{s.breadcrumb}</span>
                  <span className="ds-title">{s.title}</span>
                </button>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
