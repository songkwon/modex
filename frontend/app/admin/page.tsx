import Link from "next/link";

const links = [
  ["/admin/categories", "分类管理"],
  ["/admin/modules", "模块管理"],
  ["/admin/releases", "发布记录"],
  ["/admin/analytics", "阅读统计"],
  ["/admin/search-logs", "搜索日志"],
  ["/admin/mcp-logs", "MCP 日志"]
];

export default function AdminPage() {
  return (
    <main className="main">
      <section className="panel">
        <h1 className="text-2xl font-semibold">管理</h1>
      </section>
      <section className="card-grid mt-5">
        {links.map(([href, label]) => (
          <Link className="card" href={href} key={href}>
            <h2 className="font-semibold">{label}</h2>
            <p className="muted mt-2 text-sm">MVP 管理入口</p>
          </Link>
        ))}
      </section>
    </main>
  );
}
