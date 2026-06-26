"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Sparkles, X, Loader2, ArrowUp } from "lucide-react";
import { askAIStream } from "@/lib/api";
import type { SearchResult } from "@/types/modex";
import { useI18n } from "@/lib/i18n";
import { AiMarkdown, splitAnswerParts } from "@/components/ai-markdown";

type Message = { role: "user" | "assistant"; text: string; reasoning?: string; warning?: string; sources?: SearchResult[] };

// DocChatPanel is a right-side drawer for conversing with AI about the current
// document. Each question is answered with retrieval scoped to this module.
export function DocChatPanel({
  open,
  onClose,
  moduleKey,
  moduleName
}: {
  open: boolean;
  onClose: () => void;
  moduleKey: string;
  moduleName: string;
}) {
  const { t } = useI18n();
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bodyRef.current?.scrollTo({ top: bodyRef.current.scrollHeight, behavior: "smooth" });
  }, [messages, busy]);

  async function send() {
    const q = input.trim();
    if (!q || busy) return;
    setMessages((m) => [...m, { role: "user", text: q }]);
    setInput("");
    setBusy(true);
    try {
      const assistantIndex = messages.length + 1;
      setMessages((m) => [...m, { role: "assistant", text: "" }]);
      await askAIStream(q, { module_key: moduleKey }, (res) => {
        const parts = splitAnswerParts(res);
        setMessages((m) => m.map((msg, i) => (
          i === assistantIndex
            ? { role: "assistant", text: parts.answer, reasoning: parts.reasoning, warning: res.warning, sources: res.sources }
            : msg
        )));
      });
    } catch (e) {
      setMessages((m) => [...m, { role: "assistant", text: t("component.docChatPanel.error") + String(e) }]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div className={`doc-chat-scrim ${open ? "open" : ""}`} onClick={onClose} />
      <aside className={`doc-chat-drawer ${open ? "open" : ""}`} aria-hidden={!open}>
        <header className="doc-chat-head">
          <span className="doc-chat-title"><Sparkles size={16} className="ds-ask-icon" /> {t("nav.askAI")}</span>
          <span className="doc-chat-sub muted">{moduleName}</span>
          <button className="button icon-button" onClick={onClose} aria-label={t("component.searchResults.close")}><X size={16} /></button>
        </header>

        <div className="doc-chat-body" ref={bodyRef}>
          {messages.length === 0 ? (
            <div className="doc-chat-empty muted">
              {t("component.docChatPanel.for")}{moduleName}{t("component.docChatPanel.s_documentation_e_g")}<br />
              {t("component.docChatPanel.how_do_i_integrate_this_module_what_known")}
            </div>
          ) : null}
          {messages.map((m, i) => (
            <div key={i} className={`doc-chat-msg ${m.role}`}>
              <div className="doc-chat-bubble">
                {m.warning ? <div className="panel badge-warn" style={{ marginBottom: 8, borderRadius: 10 }}>{m.warning}</div> : null}
                {m.role === "assistant" ? <AiMarkdown answer={m.text} reasoning={m.reasoning} compact /> : m.text}
              </div>
              {m.sources && m.sources.length > 0 ? (
                <div className="doc-chat-sources">
                  {m.sources.slice(0, 4).map((src) => (
                    <Link className="doc-chat-source" key={src.doc_id} href={src.path} onClick={onClose}>
                      {src.title}
                    </Link>
                  ))}
                </div>
              ) : null}
            </div>
          ))}
          {busy ? <div className="doc-chat-msg assistant"><div className="doc-chat-bubble"><Loader2 size={15} className="ds-spin" /> {t("component.docChatPanel.thinking")}</div></div> : null}
        </div>

        <div className="doc-chat-input">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                send();
              }
            }}
            placeholder={t("component.docChatPanel.ask_questions_about_this_document")}
            rows={1}
          />
          <button className="button button-primary doc-chat-send" onClick={send} disabled={!input.trim() || busy} aria-label={t("component.docChatPanel.send")}>
            <ArrowUp size={16} />
          </button>
        </div>
      </aside>
    </>
  );
}
