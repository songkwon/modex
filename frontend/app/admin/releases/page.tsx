"use client";

import { useState } from "react";
import { History, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { usePaged } from "@/lib/use-paged";
import { useI18n } from "@/lib/i18n";

const PAGE_SIZE = 12;

type Release = {
  release_id: string;
  module_key: string;
  docs_version: string;
  branch: string;
  publisher: string;
  build_system: string;
  build_id: string;
  artifact_version: string;
  package_version: string;
  status: string;
  published_at: string;
};

export default function ReleasesPage() {
  const { t } = useI18n();
  const [keyword, setKeyword] = useState("");
  const { items: pageRows, total, page, setPage, error } = usePaged<Release>("/api/admin/releases", PAGE_SIZE, keyword.trim());

  return (
    <AdminShell title={t("legacy.7290d89a6d74")} kicker="Releases" description={t("legacy.85135e8716b5")}>
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{t("legacy.01de8216e0d0")}{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("legacy.7ff56c80c8f0")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={History}
            title={keyword ? t("legacy.f3f06af0da9a") : t("legacy.f3208952b99f")}
            hint={keyword ? t("legacy.018f0b4a413c") : t("legacy.d08269066815")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>Release</th>
                <th>{t("legacy.b07e5088eafa")}</th>
                <th>{t("legacy.78d8ccff9b0b")}</th>
                <th>{t("legacy.984d7eb384ea")}</th>
                <th>{t("legacy.e80bc10ef28b")}</th>
                <th>{t("legacy.6320b4a8722a")}</th>
                <th>{t("legacy.e8ff4d335dee")}</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.map((r) => (
                <tr key={r.release_id}>
                  <td>
                    <span className="code-chip">{r.release_id}</span>
                    <div className="muted mt-1 text-xs">{r.artifact_version}</div>
                  </td>
                  <td>{r.module_key}</td>
                  <td><span className="tag">{r.docs_version}</span></td>
                  <td>{r.publisher}</td>
                  <td>{r.build_system} #{r.build_id}</td>
                  <td><span className="badge badge-success"><span className="badge-dot" />{r.status}</span></td>
                  <td>{r.published_at?.slice(0, 10)}</td>
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
