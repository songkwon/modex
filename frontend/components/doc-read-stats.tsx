"use client";

import { useEffect, useRef, useState } from "react";
import { Eye, Loader2, LineChart as LineChartIcon, Users } from "lucide-react";
import { getDocAnalytics, type DocReadStats as Stats } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import type { TranslateFn } from "@/lib/messages";

type Tab = "trend" | "readers";
type RangeDays = 7 | 30 | 90;

// DocReadStats renders an "eye" button on a document page. Clicking it opens a
// popover with the page's reading activity: a daily read trend line chart and a
// per-reader breakdown from PostHog when configured, otherwise the built-in
// first-party analytics store.
export function DocReadStats({ docId }: { docId: string }) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<Tab>("trend");
  const [days, setDays] = useState<RangeDays>(30);
  const [data, setData] = useState<Stats | null>(null);
  const [source, setSource] = useState<"posthog" | "builtin">("builtin");
  const [error, setError] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || data) return;
    setLoading(true);
    setError(false);
    getDocAnalytics(docId, days)
      .then((res) => {
        setData(res.stats);
        setSource(res.source);
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, [open, docId, days, data]);

  useEffect(() => {
    setData(null);
  }, [docId, days]);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className="read-stats" ref={ref}>
      <button
        className="button icon-button read-stats-trigger"
        onClick={() => setOpen((v) => !v)}
        title={t("legacy.99a1d5bd3348")}
        aria-label={t("legacy.99a1d5bd3348")}
        aria-expanded={open}
      >
        <Eye size={16} />
        {data ? <span className="read-stats-total">{data.total}</span> : null}
      </button>

      {open ? (
        <div className="read-stats-pop">
          <div className="read-stats-head">
            <strong>{t("legacy.3f7111630e72")}</strong>
            <span className="tag">{source === "posthog" ? "PostHog" : t("legacy.6ddee7759454")}</span>
            <select
              className="read-stats-range"
              value={days}
              onChange={(e) => setDays(Number(e.target.value) as RangeDays)}
              aria-label={t("legacy.80ee5ade8132")}
            >
              <option value={7}>{t("legacy.2261b06712a3")}</option>
              <option value={30}>{t("legacy.f729bb3d3f7e")}</option>
              <option value={90}>{t("legacy.0909522d9092")}</option>
            </select>
            <div className="read-stats-tabs">
              <button className={`read-stats-tab${tab === "trend" ? " active" : ""}`} onClick={() => setTab("trend")}>
                <LineChartIcon size={14} /> {t("legacy.9b59e637c838")}
              </button>
              <button className={`read-stats-tab${tab === "readers" ? " active" : ""}`} onClick={() => setTab("readers")}>
                <Users size={14} /> {t("legacy.b836711928b9")}
              </button>
            </div>
          </div>

          {loading ? (
            <div className="read-stats-empty"><Loader2 size={16} className="ds-spin" /> {t("legacy.4927a53bcc88")}</div>
          ) : error ? (
            <div className="read-stats-empty muted">{t("legacy.d1d044826a45")}</div>
          ) : !data || data.total === 0 ? (
            <div className="read-stats-empty muted">{t("legacy.5d7aa3d2bc79")}</div>
          ) : tab === "trend" ? (
            <TrendChart stats={data} />
          ) : (
            <ReadersTable readers={data.readers} />
          )}
        </div>
      ) : null}
    </div>
  );
}

function TrendChart({ stats }: { stats: Stats }) {
  const { t } = useI18n();
  const { daily } = stats;
  const W = 320;
  const H = 120;
  const padX = 6;
  const padY = 12;
  const max = Math.max(1, ...daily.map((d) => d.count));
  const n = daily.length;
  const x = (i: number) => (n <= 1 ? padX : padX + (i * (W - padX * 2)) / (n - 1));
  const y = (c: number) => padY + (1 - c / max) * (H - padY * 2);
  const line = daily.map((d, i) => `${i === 0 ? "M" : "L"}${x(i).toFixed(1)},${y(d.count).toFixed(1)}`).join(" ");
  const area = `${line} L${x(n - 1).toFixed(1)},${(H - padY).toFixed(1)} L${x(0).toFixed(1)},${(H - padY).toFixed(1)} Z`;
  const first = daily[0]?.date.slice(5);
  const last = daily[n - 1]?.date.slice(5);
  const peak = daily.reduce((a, b) => (b.count > a.count ? b : a), daily[0]);

  return (
    <div className="read-stats-body">
      <div className="read-stats-summary">
        <span><strong>{stats.total}</strong><small>{t("legacy.301ded2a5a81")}</small></span>
        <span><strong>{fmtDuration(stats.avg_duration_seconds, t)}</strong><small>{t("legacy.654e27cd53c2")}</small></span>
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} className="read-stats-chart" role="img" aria-label={t("legacy.cb77993ee539")}>
        <path d={area} className="read-stats-area" />
        <path d={line} className="read-stats-line" fill="none" />
        {daily.map((d, i) =>
          d.count > 0 ? <circle key={d.date} cx={x(i)} cy={y(d.count)} r={2} className="read-stats-dot" /> : null
        )}
      </svg>
      <div className="read-stats-axis muted">
        <span>{first}</span>
        <span>{t("legacy.5fa34377ddca")} {peak?.count ?? 0}{t("legacy.bd6af0fbad9c")}</span>
        <span>{last}</span>
      </div>
    </div>
  );
}

function ReadersTable({ readers }: { readers: Stats["readers"] }) {
  const { t } = useI18n();
  if (!readers || readers.length === 0) {
    return <div className="read-stats-empty muted">{t("legacy.9b7af2319166")}</div>;
  }
  return (
    <div className="read-stats-body">
      <div className="read-stats-readers">
        {readers.map((r, i) => (
          <div className="read-stats-reader" key={`${r.user_id || r.reader}-${i}`}>
            <span className="read-stats-reader-name">{r.reader}</span>
            <span className="read-stats-reader-count">{r.count} {t("legacy.0a3ad5392111")} {fmtDuration(r.avg_duration_seconds, t)}</span>
            <span className="read-stats-reader-time muted">{fmtTime(r.last_read_at)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function fmtDuration(seconds: number, t: TranslateFn) {
  if (!Number.isFinite(seconds) || seconds <= 0) return t("legacy.3db3bfd88e0a");
  if (seconds < 60) return t("legacy.5e713c2f3240", { value1: Math.round(seconds) });
  const minutes = Math.floor(seconds / 60);
  const rest = Math.round(seconds % 60);
  return rest ? t("legacy.c0a227e60381", { value1: minutes, value2: rest }) : t("legacy.418d220ecac4", { value1: minutes });
}

function fmtTime(iso: string) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
