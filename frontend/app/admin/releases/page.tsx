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
  commit_sha: string;
  branch: string;
  publisher: string;
  build_system: string;
  build_id: string;
  trigger_type: string;
  source_ip: string;
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
    <AdminShell title={t("component.adminShell.release_history")} kicker="Releases" description={t("admin.releases.track_the_source_build_system_version_and_status")}>
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{t("admin.mcpLogs.load_failed")}{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("admin.releases.search_module_publisher_version")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
      </div>

      <div className="table-card">
        {total === 0 ? (
          <EmptyState
            icon={History}
            title={keyword ? t("admin.releases.no_matching_release_records") : t("admin.releases.no_publish_records")}
            hint={keyword ? t("admin.mcpLogs.try_a_different_keyword") : t("admin.releases.each_release_is_recorded_here_after_pushing_documentation")}
          />
        ) : (
          <div className="table-scroll"><table className="data-table">
            <thead>
              <tr>
                <th>{t("me.recent.module")}</th>
                <th>{t("admin.releases.documentation_version")}</th>
                <th>{t("admin.releases.trigger")}</th>
                <th>{t("admin.releases.source_ip")}</th>
                <th>{t("admin.releases.commit")}</th>
                <th>{t("admin.releases.status")}</th>
                <th>{t("admin.releases.published_at")}</th>
              </tr>
            </thead>
            <tbody>
              {pageRows.map((r) => (
                <tr key={r.release_id}>
                  <td>
                    <strong>{r.module_key}</strong>
                    {r.branch ? <div className="muted mt-1 text-xs">{r.branch}</div> : null}
                  </td>
                  <td>
                    <span className="tag">{r.docs_version}</span>
                    {r.package_version ? <div className="muted mt-1 text-xs">{r.package_version}</div> : null}
                  </td>
                  <td>{triggerLabel(r.trigger_type, t)}</td>
                  <td>{r.source_ip || "-"}</td>
                  <td>{r.commit_sha ? <span className="code-chip">{r.commit_sha.slice(0, 8)}</span> : <span className="muted">-</span>}</td>
                  <td><span className="badge badge-success"><span className="badge-dot" />{r.status}</span></td>
                  <td>{r.published_at ? new Date(r.published_at).toLocaleString() : "-"}</td>
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

function triggerLabel(value: string | undefined, t: (key: string) => string) {
  return value === "pipeline" ? t("admin.releases.pipeline") : t("admin.releases.manual");
}
