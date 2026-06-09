import { AdminShell } from "@/components/admin-shell";
import { api } from "@/lib/api";

type SearchLog = {
  id: string;
  user_id: string;
  query: string;
  mode: string;
  result_count: number;
  clicked_doc_id: string;
  searched_at: string;
};

export default async function SearchLogsPage() {
  const data = await api<SearchLog[]>("/api/admin/analytics/search");
  return (
    <AdminShell title="搜索日志" kicker="Search Analytics" description="用于观察高频搜索、无结果搜索词和搜索点击行为。">
      <section className="panel">
        {data.length === 0 ? (
          <div className="empty-state">
            <div>
              <div className="font-semibold text-slate-700">暂无搜索日志</div>
              <p className="mt-2 text-sm">在搜索页执行一次查询后，这里会显示记录。</p>
            </div>
          </div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>查询词</th>
                <th>模式</th>
                <th>结果数</th>
                <th>用户</th>
                <th>点击</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              {data.map((log) => (
                <tr key={log.id}>
                  <td className="font-medium">{log.query || "-"}</td>
                  <td><span className="tag">{log.mode}</span></td>
                  <td>{log.result_count}</td>
                  <td>{log.user_id || "-"}</td>
                  <td>{log.clicked_doc_id || "-"}</td>
                  <td>{log.searched_at?.slice(0, 19).replace("T", " ")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </AdminShell>
  );
}
