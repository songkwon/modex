"use client";

import { useEffect, useMemo, useState } from "react";
import { History, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { api } from "@/lib/api";

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
  const [releases, setReleases] = useState<Release[]>([]);
  const [error, setError] = useState("");
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);

  useEffect(() => {
    let cancelled = false;
    api<Release[]>("/api/admin/releases")
      .then((r) => { if (!cancelled) setReleases(r || []); })
      .catch((e) => { if (!cancelled) setError(String(e)); });
    return () => { cancelled = true; };
  }, []);

  const filtered = useMemo(() => {
    const q = keyword.trim().toLowerCase();
    if (!q) return releases;
    return releases.filter((r) =>
      (r.module_key || "").toLowerCase().includes(q) ||
      (r.publisher || "").toLowerCase().includes(q) ||
      (r.docs_version || "").toLowerCase().includes(q) ||
      (r.release_id || "").toLowerCase().includes(q),
    );
  }, [releases, keyword]);
  const pageRows = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  return (
    <AdminShell title="发布记录" kicker="Releases" description="追踪每次文档发布的来源、构建系统、版本与状态。">
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>加载失败：{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder="搜索模块 / 发布人 / 版本"
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
          />
        </div>
      </div>

      <div className="table-card">
        {filtered.length === 0 ? (
          <EmptyState
            icon={History}
            title={keyword ? "没有匹配的发布记录" : "暂无发布记录"}
            hint={keyword ? "换个关键词试试。" : "通过 CI 推送文档（docsctl deploy）后，每次发布都会记录在这里。"}
          />
        ) : (
          <>
            <div className="table-scroll"><table className="data-table">
              <thead>
                <tr>
                  <th>Release</th>
                  <th>模块</th>
                  <th>文档版本</th>
                  <th>发布人</th>
                  <th>构建</th>
                  <th>状态</th>
                  <th>发布时间</th>
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
            <Pagination page={page} pageSize={PAGE_SIZE} total={filtered.length} onPage={setPage} />
          </>
        )}
      </div>
    </AdminShell>
  );
}
