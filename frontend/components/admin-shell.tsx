"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { BarChart3, Boxes, FolderTree, History, MessageSquareText, Search, Settings, Users, UsersRound } from "lucide-react";

const adminLinks: [string, string, typeof FolderTree][] = [
  ["/admin", "概览", BarChart3],
  ["/admin/categories", "分类管理", FolderTree],
  ["/admin/teams", "团队管理", UsersRound],
  ["/admin/modules", "文档源管理", Boxes],
  ["/admin/users", "用户管理", Users],
  ["/admin/settings", "模型设置", Settings],
  ["/admin/releases", "发布记录", History],
  ["/admin/analytics", "阅读统计", BarChart3],
  ["/admin/search-logs", "搜索日志", Search],
  ["/admin/mcp-logs", "MCP 日志", MessageSquareText],
];

export function AdminShell({ title, kicker, description, children }: { title: string; kicker?: string; description?: string; children: ReactNode }) {
  const pathname = usePathname();
  return (
    <main className="main">
      <section className="admin-layout">
        <aside className="admin-nav">
          {adminLinks.map(([href, label, Icon]) => {
            const active = href === "/admin" ? pathname === "/admin" : pathname.startsWith(href);
            return (
              <Link className={`admin-nav-link${active ? " active" : ""}`} href={href} key={href}>
                <Icon size={16} />
                <span>{label}</span>
              </Link>
            );
          })}
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
