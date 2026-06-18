"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { BarChart3, Boxes, FolderTree, History, Link2, MessageCircleQuestion, MessageSquareText, Plug, Puzzle, Search, Settings, Users, UsersRound } from "lucide-react";
import { getMe } from "@/lib/api";

let cachedIsSuper: boolean | null = null;
let pendingIsSuper: Promise<boolean> | null = null;

// level "super" links are only visible to super admins; "all" links are shared
// with team admins (the team-scoped console tier).
const adminLinks: [string, string, typeof FolderTree, "super" | "all"][] = [
  ["/admin", "概览", BarChart3, "super"],
  ["/admin/categories", "分类管理", FolderTree, "all"],
  ["/admin/teams", "团队管理", UsersRound, "super"],
  ["/admin/modules", "文档源管理", Boxes, "all"],
  ["/admin/users", "用户管理", Users, "super"],
  ["/admin/settings", "模型设置", Settings, "super"],
  ["/admin/connected-apps", "应用链接", Link2, "super"],
  ["/admin/plugins", "插件管理", Plug, "super"],
  ["/admin/snippets", "复用片段", Puzzle, "super"],
  ["/admin/releases", "发布记录", History, "all"],
  ["/admin/feedback", "文档反馈", MessageCircleQuestion, "all"],
  ["/admin/search-logs", "搜索日志", Search, "all"],
  ["/admin/mcp-logs", "MCP 日志", MessageSquareText, "all"],
];

export function AdminShell({ title, kicker, description, children }: { title: string; kicker?: string; description?: string; children: ReactNode }) {
  const pathname = usePathname();
  const [isSuper, setIsSuper] = useState<boolean | null>(cachedIsSuper);
  const [pendingHref, setPendingHref] = useState<string | null>(null);

  useEffect(() => {
    if (cachedIsSuper !== null) {
      setIsSuper(cachedIsSuper);
      return;
    }

    let mounted = true;
    pendingIsSuper ??= getMe()
      .then((me) => !!me.is_super_admin)
      .catch(() => false)
      .then((value) => {
        cachedIsSuper = value;
        return value;
      })
      .finally(() => {
        pendingIsSuper = null;
      });

    pendingIsSuper.then((value) => {
      if (mounted) setIsSuper(value);
    });
    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    setPendingHref(null);
  }, [pathname]);

  // Until we know the role, show only the shared links to avoid flashing
  // super-admin-only items to team admins.
  const links = adminLinks.filter(([, , , level]) => isSuper || level === "all");
  const activePath = pendingHref || pathname;

  return (
    <main className="main">
      <section className="admin-layout">
        <aside className="admin-nav">
          {links.map(([href, label, Icon]) => {
            const active = href === "/admin" ? activePath === "/admin" : activePath.startsWith(href);
            return (
              <Link
                className={`admin-nav-link${active ? " active" : ""}`}
                href={href}
                key={href}
                onClick={(event) => {
                  if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
                  setPendingHref(href);
                }}
              >
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
