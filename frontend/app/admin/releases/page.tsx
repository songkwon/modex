import { AdminShell } from "@/components/admin-shell";
import { api } from "@/lib/api";

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

export default async function ReleasesPage() {
  const releases = await api<Release[]>("/api/admin/releases");
  return (
    <AdminShell title="发布记录" kicker="Releases" description="追踪每次文档包发布来源、构建系统、版本和发布状态。">
      <div className="table-card">
        {releases.length === 0 ? (
          <div className="empty-state" style={{ border: 0, background: "transparent" }}>暂无发布记录</div>
        ) : (
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
              {releases.map((r) => (
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
      </div>
    </AdminShell>
  );
}
