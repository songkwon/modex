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
    <AdminShell title={t("legacy.795c8bbceda7")} kicker="AI Access" description={t("legacy.7bec6afb46a2")}>
      {error ? (
        <div className="panel badge-danger" style={{ borderRadius: 12 }}>
          {t("legacy.01de8216e0d0")}{error}{t("legacy.89a00d097403")}
        </div>
      ) : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("legacy.6771285bc08c")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {loading ? (
          <div className="muted text-sm" style={{ padding: 18 }}>{t("legacy.9dc0825fba54")}</div>
        ) : total === 0 ? (
          <EmptyState
            icon={MessageSquareText}
            title={keyword ? t("legacy.e455f075d925") : t("legacy.179ffc21f317")}
            hint={keyword ? t("legacy.018f0b4a413c") : t("legacy.fea7bb8c19f5")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.5ca6730d4415")}</th>
                <th>{t("legacy.bcd6771e08ec")}</th>
                <th>{t("legacy.15c20d768e18")}</th>
                <th>{t("legacy.0d0e1a86b3aa")}</th>
                <th>{t("legacy.2087c777c06f")}</th>
                <th>{t("legacy.8b6ff498515b")}</th>
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
