"use client";

import { useState } from "react";
import { MessageCircleQuestion, Search, ThumbsDown, ThumbsUp } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { usePaged } from "@/lib/use-paged";
import { useI18n } from "@/lib/i18n";

const PAGE_SIZE = 12;

type FeedbackLog = {
  id: string;
  doc_id: string;
  module_key: string;
  title: string;
  rating: "good" | "bad";
  comment: string;
  user_id: string;
  display_name: string;
  session_id: string;
  created_at: string;
};

export default function FeedbackPage() {
  const { t } = useI18n();
  const [keyword, setKeyword] = useState("");
  const { items: pageRows, total, page, setPage, error } = usePaged<FeedbackLog>("/api/admin/analytics/feedback", PAGE_SIZE, keyword.trim());

  return (
    <AdminShell title={t("legacy.5f5b94f329bf")} kicker="Feedback" description={t("legacy.cf6af4a48e34")}>
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{t("legacy.01de8216e0d0")}{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("legacy.674e95f24014")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={MessageCircleQuestion}
            title={keyword ? t("legacy.5c1e3d7a48b8") : t("legacy.39032368a06d")}
            hint={keyword ? t("legacy.018f0b4a413c") : t("legacy.28347fba15db")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.2687ccdbb1d2")}</th>
                <th>{t("legacy.8b2106ca1371")}</th>
                <th>{t("legacy.7a688306423b")}</th>
                <th>{t("legacy.b07e5088eafa")}</th>
                <th>{t("legacy.0d0e1a86b3aa")}</th>
                <th>{t("legacy.8b6ff498515b")}</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.map((log) => (
                <tr key={log.id}>
                  <td>
                    <div className="font-medium">{log.title || log.doc_id}</div>
                    <div className="muted text-xs">{log.doc_id}</div>
                  </td>
                  <td>
                    <span className={`badge ${log.rating === "good" ? "badge-success" : "badge-warn"}`}>
                      {log.rating === "good" ? <ThumbsUp size={13} /> : <ThumbsDown size={13} />}
                      {log.rating === "good" ? t("legacy.28030e690447") : t("legacy.690d2b0654e9")}
                    </span>
                  </td>
                  <td className="muted text-sm">{log.comment || "-"}</td>
                  <td>{log.module_key || "-"}</td>
                  <td>
                    {log.display_name ? (
                      <span>{log.display_name}</span>
                    ) : log.user_id ? (
                      <span className="muted">{log.user_id}</span>
                    ) : (
                      <span className="muted">{t("legacy.34a917cd44b0")}</span>
                    )}
                  </td>
                  <td>{log.created_at?.slice(0, 19).replace("T", " ")}</td>
                </tr>
              ))}
            </tbody>
          </table></div>
        )}
        <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPage={setPage} />
      </div>
    </AdminShell>
  );
}
