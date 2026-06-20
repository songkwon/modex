"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { useI18n } from "@/lib/i18n";

/** Compact page list with first/last + ellipsis, used under admin tables. */
function pageList(current: number, total: number): (number | "…")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const out: (number | "…")[] = [1];
  const start = Math.max(2, current - 1);
  const end = Math.min(total - 1, current + 1);
  if (start > 2) out.push("…");
  for (let i = start; i <= end; i++) out.push(i);
  if (end < total - 1) out.push("…");
  out.push(total);
  return out;
}

export function Pagination({
  page,
  pageSize,
  total,
  onPage,
}: {
  page: number;
  pageSize: number;
  total: number;
  onPage: (p: number) => void;
}) {
  const { t } = useI18n();
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const to = Math.min(total, page * pageSize);

  return (
    <div className="pagination">
      <span>
        共 {total} {t("component.ui.pagination.items_page")} {from}–{to} 条
      </span>
      <div className="pagination-pages">
        <button className="page-btn" disabled={page <= 1} onClick={() => onPage(page - 1)} aria-label={t("component.ui.pagination.previous_page")}>
          <ChevronLeft size={15} />
        </button>
        {pageList(page, totalPages).map((p, i) =>
          p === "…" ? (
            <span key={`e${i}`} style={{ padding: "0 4px", color: "hsl(var(--muted))" }}>
              …
            </span>
          ) : (
            <button key={p} className={`page-btn${p === page ? " active" : ""}`} onClick={() => onPage(p)}>
              {p}
            </button>
          ),
        )}
        <button className="page-btn" disabled={page >= totalPages} onClick={() => onPage(page + 1)} aria-label={t("component.ui.pagination.next_page")}>
          <ChevronRight size={15} />
        </button>
      </div>
    </div>
  );
}
