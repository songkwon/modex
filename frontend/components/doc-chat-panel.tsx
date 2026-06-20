"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Sparkles, X, Loader2, ArrowUp } from "lucide-react";
import { askAI } from "@/lib/api";
import type { SearchResult } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

type Message = { role: "user" | "assistant"; text: string; sources?: SearchResult[] };

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
      const res = await askAI(q, { module_key: moduleKey });
      setMessages((m) => [...m, { role: "assistant", text: res.answer, sources: res.sources }]);
    } catch (e) {
      setMessages((m) => [...m, { role: "assistant", text: t("legacy.7481f4cbad36") + String(e) }]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div className={`doc-chat-scrim ${open ? "open" : ""}`} onClick={onClose} />
      <aside className={`doc-chat-drawer ${open ? "open" : ""}`} aria-hidden={!open}>
        <header className="doc-chat-head">
          <span className="doc-chat-title"><Sparkles size={16} className="ds-ask-icon" /> {t("legacy.69d8ed1c70a6")}</span>
          <span className="doc-chat-sub muted">{moduleName}</span>
          <button className="button icon-button" onClick={onClose} aria-label={t("legacy.3fd47edce45b")}><X size={16} /></button>
        </header>

        <div className="doc-chat-body" ref={bodyRef}>
          {messages.length === 0 ? (
            <div className="doc-chat-empty muted">
              {t("legacy.4f49b6aa91cc")}{moduleName}{t("legacy.9c46aa549d19")}<br />
              {t("legacy.f1a51d80d7a6")}
            </div>
          ) : null}
          {messages.map((m, i) => (
            <div key={i} className={`doc-chat-msg ${m.role}`}>
              <div className="doc-chat-bubble">{m.text}</div>
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
          {busy ? <div className="doc-chat-msg assistant"><div className="doc-chat-bubble"><Loader2 size={15} className="ds-spin" /> {t("legacy.64088d8cd78a")}</div></div> : null}
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
            placeholder={t("legacy.b01115dc1577")}
            rows={1}
          />
          <button className="button button-primary doc-chat-send" onClick={send} disabled={!input.trim() || busy} aria-label={t("legacy.edecf0ae6e51")}>
            <ArrowUp size={16} />
          </button>
        </div>
      </aside>
    </>
  );
}
