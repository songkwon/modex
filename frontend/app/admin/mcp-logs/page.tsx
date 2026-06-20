"use client";

import { useState } from "react";
import { MessageSquareText, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { usePaged } from "@/lib/use-paged";
import { useI18n } from "@/lib/i18n";

const PAGE_SIZE = 12;

type MCPLog = {
  id: string;
  tool_name: string;
  display_name: string;
  user_id: string;
  query: string;
  input_json: string;
  result_count: number;
  created_at: string;
};

export default function MCPLogsPage() {
  const { t } = useI18n();
  const [keyword, setKeyword] = useState("");
  const { items: pageRows, total, page, setPage, error, loading } = usePaged<MCPLog>("/api/admin/analytics/mcp", PAGE_SIZE, keyword.trim());

  return (
    <AdminShell title={t("component.adminShell.mcp_logs")} kicker="AI Access" description={t("admin.mcpLogs.log_ai_tool_behavior_via_mcp_reading_modules")}>
      {error ? (
        <div className="panel badge-danger" style={{ borderRadius: 12 }}>
          {t("admin.mcpLogs.load_failed")}{error}{t("admin.mcpLogs.super_admin_privileges_may_be_required")}
        </div>
      ) : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("admin.mcpLogs.search_tool_query_user")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {loading ? (
          <div className="muted text-sm" style={{ padding: 18 }}>{t("admin.mcpLogs.loading")}</div>
        ) : total === 0 ? (
          <EmptyState
            icon={MessageSquareText}
            title={keyword ? t("admin.mcpLogs.no_matching_records") : t("admin.mcpLogs.no_mcp_calls")}
            hint={keyword ? t("admin.mcpLogs.try_a_different_keyword") : t("admin.mcpLogs.after_configuring_and_connecting_an_mcp_service_ai")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.mcpLogs.tools")}</th>
                <th>{t("admin.mcpLogs.search")}</th>
                <th>{t("admin.mcpLogs.results")}</th>
                <th>{t("admin.mcpLogs.user")}</th>
                <th>{t("admin.mcpLogs.enter")}</th>
                <th>{t("admin.mcpLogs.time")}</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.map((log) => (
                <tr key={log.id}>
                  <td><span className="tag">{log.tool_name}</span></td>
                  <td className="font-medium">{log.query || "-"}</td>
                  <td>{log.result_count}</td>
                  <td>
                    {log.display_name ? (
                      <span>{log.display_name}</span>
                    ) : log.user_id ? (
                      <span className="muted">{log.user_id}</span>
                    ) : (
                      <span className="muted">-</span>
                    )}
                  </td>
                  <td><span className="code-chip">{log.input_json || "{}"}</span></td>
                  <td>{log.created_at?.slice(0, 19).replace("T", " ")}</td>
                </tr>
              ))}
            </tbody>
          </table></div>
        )}
        {!loading ? <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPage={setPage} /> : null}
      </div>
    </AdminShell>
  );
}
