"use client";

import Link from "next/link";
import { type PointerEvent, useRef, useState } from "react";
import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, GripHorizontal, Send, ThumbsDown, ThumbsUp, X } from "lucide-react";
import { capture, sessionId } from "@/lib/analytics";
import { recordDocFeedback } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

type NavLink = { title: string; href: string };
type Point = { x: number; y: number };

export function FloatingDocFeedback({ docId }: { docId: string }) {
  const { t } = useI18n();
  const [closed, setClosed] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [position, setPosition] = useState<Point>({ x: 0, y: 0 });
  const panelRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{
    pointerId: number;
    origin: Point;
    start: Point;
    bounds: { minX: number; maxX: number; minY: number; maxY: number };
  } | null>(null);

  if (closed) return null;

  const startDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0 || (event.target as HTMLElement).closest("button")) return;
    const rect = panelRef.current?.getBoundingClientRect();
    if (!rect) return;
    const baseLeft = rect.left - position.x;
    const baseTop = rect.top - position.y;
    dragRef.current = {
      pointerId: event.pointerId,
      origin: { x: event.clientX, y: event.clientY },
      start: position,
      bounds: {
        minX: 12 - baseLeft,
        maxX: window.innerWidth - rect.width - 12 - baseLeft,
        minY: 12 - baseTop,
        maxY: window.innerHeight - 44 - baseTop
      }
    };
    event.currentTarget.setPointerCapture(event.pointerId);
  };

  const drag = (event: PointerEvent<HTMLDivElement>) => {
    const current = dragRef.current;
    if (!current || current.pointerId !== event.pointerId) return;
    const x = current.start.x + event.clientX - current.origin.x;
    const y = current.start.y + event.clientY - current.origin.y;
    setPosition({
      x: Math.min(current.bounds.maxX, Math.max(current.bounds.minX, x)),
      y: Math.min(current.bounds.maxY, Math.max(current.bounds.minY, y))
    });
  };

  const stopDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (dragRef.current?.pointerId !== event.pointerId) return;
    dragRef.current = null;
    event.currentTarget.releasePointerCapture(event.pointerId);
  };

  return (
    <div
      ref={panelRef}
      className={`doc-embed-feedback${collapsed ? " is-collapsed" : ""}`}
      style={{ transform: `translate3d(${position.x}px, ${position.y}px, 0)` }}
    >
      <div
        className="doc-embed-feedback__bar"
        onPointerDown={startDrag}
        onPointerMove={drag}
        onPointerUp={stopDrag}
        onPointerCancel={stopDrag}
      >
        <GripHorizontal size={16} aria-hidden />
        <strong>{t("legacy.5f5b94f329bf")}</strong>
        <button
          type="button"
          className="icon-btn doc-embed-feedback__action"
          onClick={() => setCollapsed((value) => !value)}
          aria-label={collapsed ? t("legacy.d613dc2b47e2") : t("legacy.a5efe577647c")}
          title={collapsed ? t("legacy.d613dc2b47e2") : t("legacy.a5efe577647c")}
        >
          {collapsed ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>
        <button
          type="button"
          className="icon-btn doc-embed-feedback__action"
          onClick={() => setClosed(true)}
          aria-label={t("legacy.a7aa73da9350")}
          title={t("legacy.a7aa73da9350")}
        >
          <X size={16} />
        </button>
      </div>
      <div className="doc-embed-feedback__body">
        <DocFeedbackBox docId={docId} compact />
      </div>
    </div>
  );
}

// DocFooter shows a Fumadocs-style "How is this guide?" feedback row plus
// previous/next document cards.
export function DocFooter({
  docId,
  prev,
  next,
}: {
  docId: string;
  prev?: NavLink;
  next?: NavLink;
}) {
  const { t } = useI18n();
  return (
    <div className="doc-footer">
      <DocFeedbackBox docId={docId} />

      {prev || next ? (
        <nav className="doc-pager">
          {prev ? (
            <Link href={prev.href} className="doc-pager__card doc-pager__card--prev">
              <span className="doc-pager__dir"><ChevronLeft size={14} /> {t("legacy.5e50d5184669")}</span>
              <span className="doc-pager__title">{prev.title}</span>
            </Link>
          ) : <span />}
          {next ? (
            <Link href={next.href} className="doc-pager__card doc-pager__card--next">
              <span className="doc-pager__dir">{t("legacy.673ee5a5a418")} <ChevronRight size={14} /></span>
              <span className="doc-pager__title">{next.title}</span>
            </Link>
          ) : <span />}
        </nav>
      ) : null}
    </div>
  );
}

export function DocFeedbackBox({ docId, compact = false }: { docId: string; compact?: boolean }) {
  const { t } = useI18n();
  const [rating, setRating] = useState<"good" | "bad" | null>(null);
  const [comment, setComment] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    if (!rating || submitting) return;
    setSubmitting(true);
    capture("docs_feedback", { doc_id: docId, rating, has_comment: comment.trim().length > 0 });
    try {
      await recordDocFeedback({ doc_id: docId, rating, comment: comment.trim(), session_id: sessionId() });
      setSubmitted(true);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className={`doc-feedback${compact ? " doc-feedback--compact" : ""}`}>
      <div className="doc-feedback__top">
        <span className="doc-feedback__label">{t("legacy.e5c15acc5265")}</span>
        <div className="doc-feedback__rating" aria-label={t("legacy.3d39e4df1cc0")}>
          <button
            type="button"
            className={`doc-feedback__btn${rating === "good" ? " active" : ""}`}
            onClick={() => setRating("good")}
            disabled={submitted}
          >
            <ThumbsUp size={15} /> {t("legacy.28030e690447")}
          </button>
          <button
            type="button"
            className={`doc-feedback__btn${rating === "bad" ? " active" : ""}`}
            onClick={() => setRating("bad")}
            disabled={submitted}
          >
            <ThumbsDown size={15} /> {t("legacy.690d2b0654e9")}
          </button>
        </div>
      </div>
      {submitted ? (
        <span className="doc-feedback__thanks">{t("legacy.9d62ab8204a5")}</span>
      ) : (
        <div className="doc-feedback__form">
          <textarea
            className="doc-feedback__input"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder={t("legacy.88e629d53b5c")}
            rows={compact ? 2 : 3}
          />
          <button type="button" className="doc-feedback__submit" onClick={submit} disabled={!rating || submitting}>
            <Send size={14} /> {t("legacy.228d8b711ccf")}
          </button>
        </div>
      )}
    </div>
  );
}
