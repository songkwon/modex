"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { BarChart3, Boxes, FolderTree, History, MessageSquareText, Search, Settings, Users, UsersRound } from "lucide-react";
import { getMe } from "@/lib/api";

// level "super" links are only visible to super admins; "all" links are shared
// with team admins (the team-scoped console tier).
const adminLinks: [string, string, typeof FolderTree, "super" | "all"][] = [
  ["/admin", "概览", BarChart3, "super"],
  ["/admin/categories", "分类管理", FolderTree, "all"],
  ["/admin/teams", "团队管理", UsersRound, "super"],
  ["/admin/modules", "文档源管理", Boxes, "all"],
  ["/admin/users", "用户管理", Users, "super"],
  ["/admin/settings", "模型设置", Settings, "super"],
  ["/admin/releases", "发布记录", History, "all"],
  ["/admin/search-logs", "搜索日志", Search, "all"],
  ["/admin/mcp-logs", "MCP 日志", MessageSquareText, "all"],
];

export function AdminShell({ title, kicker, description, children }: { title: string; kicker?: string; description?: string; children: ReactNode }) {
  const pathname = usePathname();
  const [isSuper, setIsSuper] = useState<boolean | null>(null);
  useEffect(() => {
    getMe().then((me) => setIsSuper(!!me.is_super_admin)).catch(() => setIsSuper(false));
  }, []);
  // Until we know the role, show only the shared links to avoid flashing
  // super-admin-only items to team admins.
  const links = adminLinks.filter(([, , , level]) => isSuper || level === "all");
  return (
    <main className="main">
      <section className="admin-layout">
        <aside className="admin-nav">
          {links.map(([href, label, Icon]) => {
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
