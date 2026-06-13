"use client";

import { useEffect, useRef, useState } from "react";
import { Eye, Loader2, LineChart as LineChartIcon, Users } from "lucide-react";
import { getDocAnalytics, type DocReadStats as Stats } from "@/lib/api";

type Tab = "trend" | "readers";

// DocReadStats renders an "eye" button on a document page. Clicking it opens a
// popover with the page's reading activity: a daily read trend line chart and a
// per-reader breakdown. Data comes from PostHog when the server is configured
// for it, otherwise from the built-in page-view store.
export function DocReadStats({ docId }: { docId: string }) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<Tab>("trend");
  const [data, setData] = useState<Stats | null>(null);
  const [source, setSource] = useState<string>("");
  const [error, setError] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || data) return;
    setLoading(true);
    setError(false);
    getDocAnalytics(docId)
      .then((res) => {
        setData(res.stats);
        setSource(res.source);
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, [open, docId, data]);

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
        title="查看阅读情况"
        aria-label="查看阅读情况"
        aria-expanded={open}
      >
        <Eye size={16} />
        {data ? <span className="read-stats-total">{data.total}</span> : null}
      </button>

      {open ? (
        <div className="read-stats-pop">
          <div className="read-stats-head">
            <strong>阅读情况</strong>
            <span className="tag">{source === "posthog" ? "PostHog" : "内置统计"}</span>
            <div className="read-stats-tabs">
              <button className={`read-stats-tab${tab === "trend" ? " active" : ""}`} onClick={() => setTab("trend")}>
                <LineChartIcon size={14} /> 趋势
              </button>
              <button className={`read-stats-tab${tab === "readers" ? " active" : ""}`} onClick={() => setTab("readers")}>
                <Users size={14} /> 阅读者
              </button>
            </div>
          </div>

          {loading ? (
            <div className="read-stats-empty"><Loader2 size={16} className="ds-spin" /> 加载中…</div>
          ) : error ? (
            <div className="read-stats-empty muted">加载失败</div>
          ) : !data || data.total === 0 ? (
            <div className="read-stats-empty muted">暂无阅读记录</div>
          ) : tab === "trend" ? (
            <TrendChart daily={data.daily} />
          ) : (
            <ReadersTable readers={data.readers} />
          )}
        </div>
      ) : null}
    </div>
  );
}

function TrendChart({ daily }: { daily: Stats["daily"] }) {
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
      <svg viewBox={`0 0 ${W} ${H}`} className="read-stats-chart" role="img" aria-label="每日阅读量趋势">
        <path d={area} className="read-stats-area" />
        <path d={line} className="read-stats-line" fill="none" />
        {daily.map((d, i) =>
          d.count > 0 ? <circle key={d.date} cx={x(i)} cy={y(d.count)} r={2} className="read-stats-dot" /> : null
        )}
      </svg>
      <div className="read-stats-axis muted">
        <span>{first}</span>
        <span>峰值 {peak?.count ?? 0}/日</span>
        <span>{last}</span>
      </div>
    </div>
  );
}

function ReadersTable({ readers }: { readers: Stats["readers"] }) {
  if (!readers || readers.length === 0) {
    return <div className="read-stats-empty muted">暂无阅读者</div>;
  }
  return (
    <div className="read-stats-body">
      <div className="read-stats-readers">
        {readers.map((r, i) => (
          <div className="read-stats-reader" key={`${r.user_id || r.reader}-${i}`}>
            <span className="read-stats-reader-name">{r.reader}</span>
            <span className="read-stats-reader-count">{r.count} 次</span>
            <span className="read-stats-reader-time muted">{fmtTime(r.last_read_at)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function fmtTime(iso: string) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
