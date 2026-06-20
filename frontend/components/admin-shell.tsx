"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { BarChart3, Boxes, FolderTree, History, Link2, MessageCircleQuestion, MessageSquareText, Plug, Puzzle, Search, Settings, Users, UsersRound } from "lucide-react";
import { getMe } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

let cachedIsSuper: boolean | null = null;
let pendingIsSuper: Promise<boolean> | null = null;

export function AdminShell({
  title,
  kicker,
  description,
  contentClassName,
  children,
}: {
  title: string;
  kicker?: string;
  description?: string;
  contentClassName?: string;
  children: ReactNode;
}) {
  const { t } = useI18n();
  const pathname = usePathname();
  // level "super" links are only visible to super admins; "all" links are shared
  // with team admins (the team-scoped console tier).
  const adminLinks: [string, string, typeof FolderTree, "super" | "all"][] = [
    ["/admin", t("legacy.fea405f9b01d"), BarChart3, "super"],
    ["/admin/categories", t("legacy.62f8edc30321"), FolderTree, "all"],
    ["/admin/teams", t("legacy.95a792201917"), UsersRound, "super"],
    ["/admin/modules", t("legacy.608c2486d555"), Boxes, "all"],
    ["/admin/users", t("legacy.fbf413d429bd"), Users, "super"],
    ["/admin/settings", t("legacy.082a738ae620"), Settings, "super"],
    ["/admin/connected-apps", t("legacy.213f5505bd1a"), Link2, "super"],
    ["/admin/plugins", t("legacy.35fcbd57d58a"), Plug, "super"],
    ["/admin/snippets", t("legacy.149f545ffeb8"), Puzzle, "super"],
    ["/admin/releases", t("legacy.7290d89a6d74"), History, "all"],
    ["/admin/feedback", t("legacy.5f5b94f329bf"), MessageCircleQuestion, "all"],
    ["/admin/search-logs", t("legacy.1b9b75f51d20"), Search, "all"],
    ["/admin/mcp-logs", t("legacy.795c8bbceda7"), MessageSquareText, "all"],
  ];
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
        <div className={`grid${contentClassName ? ` ${contentClassName}` : ""}`}>
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
