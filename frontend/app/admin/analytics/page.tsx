import { getPageAnalytics } from "@/lib/api";
import { AdminShell } from "@/components/admin-shell";

export default async function AnalyticsPage() {
  const data = await getPageAnalytics();
  return (
    <AdminShell title="阅读统计" kicker="Analytics" description="观察文档 PV、UV、近 7 天阅读和平均阅读时长，后续可接 PostHog 和数据库聚合。">
      <section className="panel">
        <div className="metric-strip !mt-0 text-sm">
          <div className="metric">
            <div className="muted">总 PV</div>
            <div className="text-xl font-semibold">{data.total_pv}</div>
          </div>
          <div className="metric">
            <div className="muted">近 7 天阅读</div>
            <div className="text-xl font-semibold">{data.reads_7d}</div>
          </div>
          <div className="metric">
            <div className="muted">页面数</div>
            <div className="text-xl font-semibold">{data.popular_pages.length}</div>
          </div>
          <div className="metric">
            <div className="muted">事件类型</div>
            <div className="text-xl font-semibold">{data.events.length}</div>
          </div>
        </div>

        <div className="mt-6 overflow-auto">
          <table className="data-table">
            <thead>
              <tr>
                <th className="py-2 pr-4">文档</th>
                <th className="py-2 pr-4">模块</th>
                <th className="py-2 pr-4">版本</th>
                <th className="py-2 pr-4">PV</th>
                <th className="py-2 pr-4">UV</th>
                <th className="py-2 pr-4">近7天</th>
                <th className="py-2 pr-4">近30天</th>
                <th className="py-2 pr-4">平均时长(s)</th>
              </tr>
            </thead>
            <tbody>
              {data.popular_pages.map((p) => (
                <tr key={p.doc_id}>
                  <td>
                    <a className="font-medium" href={p.path}>{p.title}</a>
                    <div className="muted text-xs">{p.doc_id}</div>
                  </td>
                  <td>{p.module_name}</td>
                  <td>{p.docs_version}</td>
                  <td>{p.pv}</td>
                  <td>{p.uv}</td>
                  <td>{p.reads_7d}</td>
                  <td>{p.reads_30d}</td>
                  <td>{p.avg_duration_seconds}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <p className="muted mt-4 text-xs">埋点事件：{data.events.join(" / ")}</p>
      </section>
    </AdminShell>
  );
}
