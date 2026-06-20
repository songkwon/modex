"use client";

import { useState } from "react";
import { Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { usePaged } from "@/lib/use-paged";
import { useI18n } from "@/lib/i18n";

const PAGE_SIZE = 12;

type SearchLog = {
  id: string;
  user_id: string;
  display_name: string;
  ip_address: string;
  query: string;
  mode: string;
  result_count: number;
  clicked_doc_id: string;
  searched_at: string;
};

export default function SearchLogsPage() {
  const { t } = useI18n();
  const [keyword, setKeyword] = useState("");
  const { items: pageRows, total, page, setPage, error } = usePaged<SearchLog>("/api/admin/analytics/search", PAGE_SIZE, keyword.trim());

  return (
    <AdminShell title={t("component.adminShell.search_log")} kicker="Search Analytics" description={t("admin.searchLogs.monitor_high_frequency_search_terms_zero_result_queries")}>
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{t("admin.mcpLogs.load_failed")}{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("admin.searchLogs.search_query_user_ip")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={Search}
            title={keyword ? t("admin.searchLogs.no_matching_search_logs") : t("admin.searchLogs.no_search_logs")}
            hint={keyword ? t("admin.mcpLogs.try_a_different_keyword") : t("admin.searchLogs.records_appear_here_after_a_user_performs_a")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.searchLogs.search_term")}</th>
                <th>{t("admin.searchLogs.mode")}</th>
                <th>{t("admin.mcpLogs.results")}</th>
                <th>{t("admin.mcpLogs.user")}</th>
                <th>{t("admin.searchLogs.click")}</th>
                <th>{t("admin.mcpLogs.time")}</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.map((log) => (
                <tr key={log.id}>
                  <td className="font-medium">{log.query || "-"}</td>
                  <td><span className="tag">{log.mode}</span></td>
                  <td>{log.result_count}</td>
                  <td>
                    {log.display_name ? (
                      <span>{log.display_name}</span>
                    ) : log.user_id ? (
                      <span className="muted">{log.user_id}</span>
                    ) : log.ip_address ? (
                      <span className="muted">{log.ip_address}</span>
                    ) : (
                      <span className="muted">-</span>
                    )}
                  </td>
                  <td>{log.clicked_doc_id || "-"}</td>
                  <td>{log.searched_at?.slice(0, 19).replace("T", " ")}</td>
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
