import Link from "next/link";
import { ArrowRight, Boxes, FolderTree, History, Link2, MessageCircleQuestion, MessageSquareText, Search, Settings, Users, UsersRound } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { ReindexControls } from "@/components/reindex-controls";
import { getServerI18n } from "@/lib/i18n-server";

export default async function AdminPage() {
  const { t } = await getServerI18n();
  const links = [
    ["/admin/categories", t("legacy.62f8edc30321"), t("legacy.c379199f21fd"), FolderTree],
    ["/admin/teams", t("legacy.95a792201917"), t("legacy.93a61c1faea7"), UsersRound],
    ["/admin/modules", t("legacy.79963951e270"), t("legacy.edfc0242df6d"), Boxes],
    ["/admin/users", t("legacy.fbf413d429bd"), t("legacy.825a96349864"), Users],
    ["/admin/settings", t("legacy.082a738ae620"), t("legacy.ca45f732b35e"), Settings],
    ["/admin/connected-apps", t("legacy.213f5505bd1a"), t("legacy.7a58eb329136"), Link2],
    ["/admin/releases", t("legacy.7290d89a6d74"), t("legacy.cf320f403b63"), History],
    ["/admin/feedback", t("legacy.5f5b94f329bf"), t("legacy.44bc7f387b74"), MessageCircleQuestion],
    ["/admin/search-logs", t("legacy.1b9b75f51d20"), t("legacy.6a460742faa7"), Search],
    ["/admin/mcp-logs", t("legacy.795c8bbceda7"), t("legacy.baa3b6782ce5"), MessageSquareText],
  ] as const;
  return (
    <AdminShell title={t("legacy.87354adb1c31")} kicker="Admin" description={t("legacy.517da0e94f26")}>
      <section className="card-grid">
        {links.map(([href, label, desc, Icon]) => (
          <Link className="admin-entry" href={href} key={href}>
            <span className="admin-entry-icon">
              <Icon size={20} />
            </span>
            <div className="admin-entry-body">
              <h2>{label}</h2>
              <p>{desc}</p>
            </div>
            <ArrowRight size={16} className="admin-entry-arrow" />
          </Link>
        ))}
      </section>
      <ReindexControls />
    </AdminShell>
  );
}
