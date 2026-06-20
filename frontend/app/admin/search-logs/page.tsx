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
    <AdminShell title={t("legacy.1b9b75f51d20")} kicker="Search Analytics" description={t("legacy.9f4505ae6160")}>
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{t("legacy.01de8216e0d0")}{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("legacy.77c5e12f55eb")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={Search}
            title={keyword ? t("legacy.4d4e33d46e72") : t("legacy.2c45f43d5445")}
            hint={keyword ? t("legacy.018f0b4a413c") : t("legacy.705c85d8bb95")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.97e406603ba9")}</th>
                <th>{t("legacy.47a270081ab2")}</th>
                <th>{t("legacy.15c20d768e18")}</th>
                <th>{t("legacy.0d0e1a86b3aa")}</th>
                <th>{t("legacy.dc71485f8ee9")}</th>
                <th>{t("legacy.8b6ff498515b")}</th>
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
