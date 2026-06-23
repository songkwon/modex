"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { getModule, rotateDeployToken } from "@/lib/api";
import type { ModuleInfo } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

export default function AdminModuleDetail({ params }: { params: Promise<{ moduleKey: string }> }) {
  const { t } = useI18n();
  const [module, setModule] = useState<ModuleInfo | null>(null);
  const [shownToken, setShownToken] = useState<string>("");
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load(key: string) {
    setError(null);
    try {
      const m = await getModule(key);
      setModule(m);
    } catch (e) {
      setError(t("admin.modules.module.failed_to_load_module_information") + (e instanceof Error ? e.message : String(e)));
      setModule(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let cancelled = false;
    params.then(({ moduleKey: key }) => {
      if (cancelled) return;
      load(key);
    });
    return () => {
      cancelled = true;
    };
  }, [params]);

  if (loading) {
    return <AdminShell title={t("admin.mcpLogs.loading")} kicker="Module Detail"><div className="panel">{t("admin.modules.module.loading_module_information")}</div></AdminShell>;
  }

  if (error || !module) {
    return (
      <AdminShell title={t("admin.modules.module.error")} kicker="Module Detail">
        <div className="panel text-red-600">{error || t("admin.modules.module.module_not_found")}</div>
      </AdminShell>
    );
  }

  const rows = [
    [t("admin.modules.module.module_key"), module.module_key],
    [t("admin.categories.description"), module.description],
    [t("common.category"), module.category_path],
    [t("admin.modules.module.owner"), module.owner_group],
    [t("admin.modules.module.maintainer"), module.maintainers?.join(", ") || "-"],
    [t("component.moduleDrawer.default_document_version"), module.default_version],
    [t("admin.modules.module.engineering_version"), module.package_version],
    [t("admin.modules.module.channel"), module.channel],
    [t("admin.modules.module.edition"), module.edition],
    [t("admin.modules.module.source_repository"), module.repo_url]
  ];

  const gitlabRows = [
    [t("admin.modules.module.sourceType"), module.source_type || "manual"],
    [t("admin.modules.module.gitlabBranch"), module.gitlab_branch || "-"],
    [t("admin.modules.module.gitlabDocsPath"), module.gitlab_path || "docs (root)"],
    [t("admin.modules.module.lastSyncedCommit"), module.last_synced_commit || "-"],
    [t("admin.modules.module.lastSynced"), module.last_synced_at || "-"],
  ];

  async function handleRotateDeployToken() {
    if (!module) return;
    if (!confirm(t("admin.modules.module.are_you_sure_you_want_to_generate_a"))) return;
    try {
      const res = await rotateDeployToken(module.module_key);
      setShownToken(res.deploy_token);
      setCopied(false);
      alert(t("admin.modules.module.a_new_deploy_token_has_been_generated_and"));
      await load(module.module_key);
    } catch (e) {
      alert(t("admin.modules.module.generation_failed") + (e instanceof Error ? e.message : String(e)));
    }
  }

  async function copyToken() {
    if (!shownToken) return;
    try {
      await navigator.clipboard.writeText(shownToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      alert(t("admin.modules.module.copy_failed_please_manually_select_and_copy_the"));
    }
  }

  return (
    <AdminShell title={module.name} kicker="Module Detail" description={t("admin.modules.module.module_governance_fields_version_entry_points_and_subsequent")}>
      <section className="split-panel">
        <div className="panel">
          <div className="section-heading">
            <h2>{t("admin.modules.module.governance_field")}</h2>
            <span className="tag">{module.status}</span>
          </div>
          <dl className="detail-list">
            {rows.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd className={value === module.repo_url ? "break-all" : ""}>{value}</dd>
              </div>
            ))}
          </dl>
        </div>
        <div className="panel">
          <div className="section-heading">
            <h2>{t("admin.modules.module.tag")}</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            {module.keywords?.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
          </div>
          <div className="empty-state mt-5">
            <div>
              <div className="font-semibold text-foreground">{t("admin.modules.module.version_and_permission_management")}</div>
              <p className="mt-2 text-sm">{t("admin.modules.module.manage_default_version_publish_permissions_and_read_permissions")}</p>
            </div>
          </div>
        </div>
      </section>

      {/* GitLab 集成 & 部署 */}
      <section className="panel">
        <div className="section-heading">
          <h2>{t("admin.modules.module.gitlab_integration_and_deployment")}</h2>
          <span className="tag">{t("admin.modules.module.ci_driven")}</span>
        </div>

        <div className="mt-4">
          <h3 className="font-medium mb-2">{t("admin.modules.module.source_configuration")}</h3>
          <dl className="detail-list text-sm">
            {gitlabRows.map(([label, value]) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
        </div>

        {/* Deploy Token Management - direct in UI */}
        <div className="mt-6">
          <h3 className="font-medium mb-2">{t("admin.modules.module.deploy_token_for_secure_gitlab_ci_deployments")}</h3>
          <div className="flex items-center gap-2">
            <button className="button button-primary" onClick={handleRotateDeployToken}>
              {t("admin.modules.module.generate_rotate_token")}
            </button>
          </div>
          {shownToken && (
            <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <strong>{t("admin.modules.module.new_token_displayed_only_once_copy_immediately_to")}</strong>
                </div>
                <button
                  className="shrink-0 px-2 py-1 text-xs rounded border border-yellow-300 bg-white hover:bg-yellow-100 transition"
                  onClick={copyToken}
                >
                  {copied ? t("component.ui.copyButton.copied") : t("component.ui.copyButton.copy")}
                </button>
              </div>
              <pre className="mt-2 p-2 bg-panel font-mono text-xs break-all select-all">{shownToken}</pre>
            </div>
          )}
          <p className="text-xs muted mt-2">
            {t("admin.modules.module.sent_in_ci_using_the_x_modex_deploy")}
            {module?.deploy_token_set
              ? t("admin.modules.module.a_deploy_token_is_already_configured_plaintext_is")
              : t("admin.modules.module.no_deploy_token_is_currently_configured")}
          </p>
        </div>

        <div className="mt-6">
          <h3 className="font-medium mb-2">{t("admin.modules.module.recommended_integration_method_similar_to_mintlify")}</h3>
          <div className="prose prose-sm max-w-none text-sm">
            <p>{t("admin.modules.module.recommended_in")}<strong>{t("admin.modules.module.gitlab_ci_for_documentation_repository")}</strong> {t("admin.modules.module.in")} <code>docsctl deploy</code>{t("admin.modules.module.automatically_performs_verification_building_packaging_and_uploading")}</p>
            <ul>
              <li>{t("admin.modules.module.compilation_completes_in_the_source_repository_environment_correct")}</li>
              <li>{t("admin.modules.module.modex_accepts_only_pre_built_static_sites_structured")}</li>
              <li>{t("admin.modules.module.a_repository_can_deploy_multiple_modules_different_subdirectories")}</li>
            </ul>
          </div>

          <div className="mt-4 p-4 bg-slate-50 rounded text-xs font-mono overflow-auto">
            <div className="mb-2 text-slate-500"># .gitlab-ci.yml 示例 (rd-doc 仓库)</div>
            <pre>{`include:
  - remote: "https://raw.githubusercontent.com/songkwon/modex/main/deploy/ci/modex-docs.gitlab-ci.yml"

variables:
  MODEX_MODULE_KEY: "rd-standards"
  # 只有多目录仓库才需要：
  # DOCS_SOURCE_DIR: "docs/standard"
  MODEX_DEPLOY_URL: "https://modex.example.com/api/deploy"

# DOCS_DEPLOY_TOKEN: 从上方生成并复制到 GitLab CI Variables（Masked + Protected）`}</pre>
          </div>

          <p className="mt-3 text-sm text-slate-600">
            {t("admin.modules.module.after_deployment_documents_are_automatically_assigned_to_the")}
          </p>

          <div className="mt-4 text-xs">
            <strong>{t("admin.modules.module.rd_doc_example")}</strong>{t("admin.modules.module.repository_root_contains_docs_standard_docs_tools_version")}<br/>
            {t("admin.modules.module.can_be_deployed_separately_as_different_jobs_or")} <code>rd-standard</code>、<code>rd-version-control</code> {t("admin.modules.module.modules_then_assign_them_to_corresponding_domains_e")}
          </div>
        </div>
      </section>
    </AdminShell>
  );
}
