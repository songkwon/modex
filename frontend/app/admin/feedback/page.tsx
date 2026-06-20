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
    <AdminShell title={t("component.adminShell.documentation_feedback")} kicker="Feedback" description={t("admin.feedback.view_helpful_needs_improvement_feedback_submitted_by_readers")}>
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{t("admin.mcpLogs.load_failed")}{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("admin.feedback.search_doc_module_user_feedback")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={MessageCircleQuestion}
            title={keyword ? t("admin.feedback.no_matching_feedback") : t("admin.feedback.no_document_feedback")}
            hint={keyword ? t("admin.mcpLogs.try_a_different_keyword") : t("admin.feedback.records_appear_here_after_a_user_clicks_the")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("me.recent.documentation")}</th>
                <th>{t("admin.feedback.feedback")}</th>
                <th>{t("admin.feedback.content")}</th>
                <th>{t("me.recent.module")}</th>
                <th>{t("admin.mcpLogs.user")}</th>
                <th>{t("admin.mcpLogs.time")}</th>
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
                      {log.rating === "good" ? t("component.docFooter.helpful") : t("component.docFooter.needs_improvement")}
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
                      <span className="muted">{t("admin.feedback.anonymous")}</span>
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
