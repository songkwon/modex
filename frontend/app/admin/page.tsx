import Link from "next/link";
import { ArrowRight, Boxes, FolderTree, History, Link2, MessageCircleQuestion, MessageSquareText, Search, Settings, Users, UsersRound } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { ReindexControls } from "@/components/reindex-controls";
import { getServerI18n } from "@/lib/i18n-server";

export default async function AdminPage() {
  const { t } = await getServerI18n();
  const links = [
    ["/admin/categories", t("component.adminShell.category_management"), t("admin.maintain_hierarchy_categories_ordering_status_and_responsible_teams"), FolderTree],
    ["/admin/teams", t("component.adminShell.team_management"), t("admin.manage_the_documentation_team_leads_and_members_and"), UsersRound],
    ["/admin/modules", t("admin.documentation_source"), t("admin.connect_documentation_repository_bind_categories_generate_deploy_token"), Boxes],
    ["/admin/users", t("component.adminShell.user_management"), t("admin.manage_user_profiles_statuses_and_permissions"), Users],
    ["/admin/settings", t("component.adminShell.model_settings"), t("admin.integrate_chat_vector_re_ranking_models_to_enable"), Settings],
    ["/admin/connected-apps", t("component.adminShell.app_link"), t("admin.register_external_applications_to_access_modex_api_and"), Link2],
    ["/admin/releases", t("component.adminShell.release_history"), t("admin.track_the_source_build_and_status_of_each"), History],
    ["/admin/feedback", t("component.adminShell.documentation_feedback"), t("admin.view_reader_submitted_helpful_needs_improvement_feedback"), MessageCircleQuestion],
    ["/admin/search-logs", t("component.adminShell.search_log"), t("admin.analyze_query_terms_hits_and_clicks"), Search],
    ["/admin/mcp-logs", t("component.adminShell.mcp_logs"), t("admin.view_ai_tool_document_reading_history"), MessageSquareText],
  ] as const;
  return (
    <AdminShell title={t("admin.admin_console")} kicker="Admin" description={t("admin.manage_categories_teams_users_document_sources_models_and")}>
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
