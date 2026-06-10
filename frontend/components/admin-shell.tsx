import Link from "next/link";
import type { ReactNode } from "react";

const adminLinks = [
  ["/admin", "概览"],
  ["/admin/categories", "分类管理"],
  ["/admin/teams", "团队管理"],
  ["/admin/modules", "模块管理"],
  ["/admin/users", "用户管理"],
  ["/admin/releases", "发布记录"],
  ["/admin/analytics", "阅读统计"],
  ["/admin/search-logs", "搜索日志"],
  ["/admin/mcp-logs", "MCP 日志"]
];

export function AdminShell({ title, kicker, description, children }: { title: string; kicker?: string; description?: string; children: ReactNode }) {
  return (
    <main className="main">
      <section className="admin-layout">
        <aside className="panel admin-nav">
          <div className="mb-3 px-2">
            <div className="brand">
              <span className="brand-mark">M</span>
              <span>Admin</span>
            </div>
          </div>
          {adminLinks.map(([href, label]) => (
            <Link className="category-node" href={href} key={href}>
              <span>{label}</span>
            </Link>
          ))}
        </aside>
        <div className="grid">
          <header className="hero-panel">
            {kicker ? <div className="page-kicker">{kicker}</div> : null}
            <h1 className="page-title">{title}</h1>
            {description ? <p className="hero-copy">{description}</p> : null}
          </header>
          {children}
        </div>
      </section>
    </main>
  );
}
