"use client";

import { useEffect, useMemo, useState } from "react";
import { BookOpen, Boxes, GitBranch, Info, Pencil, Plus, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { CopyButton } from "@/components/ui/copy-button";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { Mermaid } from "@/components/mdx/mermaid";
import { createModule, getManagedCategories, getDeployToken, rotateDeployToken, updateModule } from "@/lib/api";
import { gitlabCiTemplateInclude } from "@/lib/runtime-config";
import { usePaged } from "@/lib/use-paged";
import type { Category, ModuleInfo } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

const PAGE_SIZE = 8;
const COMPILED = new Set(["vitepress", "vuepress", "fumadocs", "docusaurus", "mkdocs", "honkit", "gitbook", "static"]);
const INTEGRATION_FLOW = `sequenceDiagram
    autonumber
    participant User as 管理员
    participant Modex as Modex
    participant CI as 文档仓库 CI
    participant Docsctl as docsctl
    User->>Modex: 创建文档源并复制 Deploy Token
    User->>CI: 配置 Deploy Token 和构建参数
    CI->>Docsctl: 执行 docsctl deploy
    Docsctl->>Docsctl: 注入 DOCS_BASE 并构建、打包
    Docsctl->>Modex: 上传文档制品和仓库元数据
    Modex-->>User: 归档版本并开放检索`;

const BASE_CONFIGS = [
  {
    name: "VitePress",
    file: "docs/.vitepress/config.ts",
    code: `export default defineConfig({\n  base: process.env.DOCS_BASE || "/",\n})`,
  },
  {
    name: "VuePress",
    file: "docs/.vuepress/config.ts",
    code: `export default defineUserConfig({\n  base: process.env.DOCS_BASE || "/",\n})`,
  },
  {
    name: "Docusaurus",
    file: "docusaurus.config.ts",
    code: `export default {\n  baseUrl: process.env.DOCS_BASE || "/",\n}`,
  },
  {
    name: "Fumadocs / Next.js",
    file: "next.config.mjs",
    code: `const basePath = (process.env.DOCS_BASE || "").replace(/\\\/$/, "");\nexport default { basePath, assetPrefix: basePath || undefined, output: "export" };`,
  },
  {
    name: "MkDocs",
    file: "mkdocs.yml",
    code: `site_url: !ENV [DOCS_BASE, "/"]`,
  },
];

type Draft = {
  module_key?: string;
  name: string;
  doc_type: string;
  mount: string;
  category_ids: string[];
  description: string;
  // read-only source metadata (filled by docsctl deploy from CI)
  repo_url?: string;
  repo_type?: string;
  gitlab_branch?: string;
};

const emptyDraft: Draft = { name: "", doc_type: "vitepress", mount: "single", category_ids: [], description: "" };

function maskToken(t: string) {
  if (!t) return "";
  if (t.length <= 10) return "•".repeat(t.length);
  return `${t.slice(0, 6)}${"•".repeat(Math.max(8, t.length - 10))}${t.slice(-4)}`;
}

function flatten(cats: Category[], depth = 0): ComboOption[] {
  return cats.flatMap((c) => [{ value: c.id, label: c.name, hint: c.key, depth }, ...flatten(c.children || [], depth + 1)]);
}

function localDeployCommand(deployUrl?: string) {
  return [
    "docsctl deploy \\",
    "  --source /path/to/docs \\",
    "  --version latest \\",
    `  --deploy-url ${deployUrl || "https://modex.example.com/api/deploy"} \\`,
    "  --token $MODEX_DEPLOY_TOKEN",
  ].join("\n");
}

function gitlabSnippet(deployUrl: string | undefined, templateInclude: string) {
  return `${templateInclude}

variables:
  MODEX_DEPLOY_URL: "${deployUrl || "https://modex.example.com/api/deploy"}"

# 在 GitLab Settings > CI/CD > Variables 中添加：
# MODEX_DEPLOY_TOKEN = 从 Modex 文档源复制的 Deploy Token（Masked + Protected）`;
}

export default function AdminModulesPage() {
  const { t } = useI18n();
  const DOC_TYPES = [
    { value: "vitepress", label: "VitePress", hint: t("admin.modules.compiled_static_site") },
    { value: "vuepress", label: "VuePress", hint: t("admin.modules.compiled") },
    { value: "fumadocs", label: "Fumadocs", hint: t("admin.modules.compiled_next_mdx") },
    { value: "docusaurus", label: "Docusaurus", hint: t("admin.modules.compiled_static_site") },
    { value: "mkdocs", label: "MkDocs", hint: t("admin.modules.compiled_python") },
    { value: "honkit", label: "HonKit / GitBook", hint: t("admin.modules.compiled_static_site") },
    { value: "markdown", label: "Markdown", hint: t("admin.modules.modex_rendering") },
    { value: "static", label: "Static", hint: t("admin.modules.generic_static_site") },
  ];
  const [categories, setCategories] = useState<Category[]>([]);
  const [keyword, setKeyword] = useState("");
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [guideOpen, setGuideOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [token, setToken] = useState<{ deploy_token: string; deploy_url: string } | null>(null);
  const [templateInclude, setTemplateInclude] = useState(() => gitlabCiTemplateInclude());
  const isEdit = !!draft.module_key;

  const { items: pageItems, total, page, setPage, error: loadError, reload } = usePaged<ModuleInfo>(
    "/api/admin/modules",
    PAGE_SIZE,
    keyword.trim(),
  );

  useEffect(() => {
    setTemplateInclude(gitlabCiTemplateInclude());
    getManagedCategories().then((tree) => setCategories(tree || [])).catch(() => {});
  }, []);

  const categoryOptions = useMemo(() => flatten(categories), [categories]);

  function openCreate() {
    setDraft(emptyDraft);
    setToken(null);
    setModalOpen(true);
  }
  function openEdit(m: ModuleInfo) {
    setDraft({
      module_key: m.module_key,
      name: m.name,
      doc_type: m.doc_type || "vitepress",
      mount: m.mount || "single",
      category_ids: m.category_ids || [],
      description: m.description || "",
      repo_url: m.repo_url,
      repo_type: m.repo_type,
      gitlab_branch: m.gitlab_branch,
    });
    setToken(null);
    setModalOpen(true);
    getDeployToken(m.module_key).then(setToken).catch(() => {});
  }

  function patchFromDraft(): Partial<ModuleInfo> {
    const compiled = COMPILED.has(draft.doc_type);
    return {
      name: draft.name,
      doc_type: draft.doc_type,
      mount: compiled ? "single" : draft.mount,
      category_ids: draft.category_ids,
      description: draft.description,
    };
  }

  async function submit() {
    setError("");
    try {
      if (isEdit) {
        await updateModule(draft.module_key!, patchFromDraft());
        setModalOpen(false);
      } else {
        const created = await createModule(patchFromDraft());
        const t = await getDeployToken(created.module_key).catch(() => null);
        setDraft({ ...draft, module_key: created.module_key });
        setToken(t);
      }
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  async function rotate() {
    if (!draft.module_key) return;
    if (!confirm(t("admin.modules.regenerate_deploy_token_the_old_token_will_expire"))) return;
    try {
      setToken(await rotateDeployToken(draft.module_key));
    } catch (e) {
      setError(String(e));
    }
  }

  const compiled = COMPILED.has(draft.doc_type);

  return (
    <AdminShell title={t("component.adminShell.document_source_management")} kicker="Doc Sources" description={t("admin.modules.connect_git_svn_documentation_repository_select_documentation_framework")}>
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input placeholder={t("admin.modules.search_name_key_repo")} value={keyword} onChange={(e) => setKeyword(e.target.value)} />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button" onClick={() => setGuideOpen(true)}><BookOpen size={16} /> {t("admin.modules.integrationGuideAction")}</button>
          <button className="button button-primary" onClick={openCreate}><Plus size={16} /> {t("admin.modules.connectDocSource")}</button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.documentation_source")}</th>
                <th>{t("common.category")}</th>
                <th>{t("admin.modules.framework_mount")}</th>
                <th>{t("admin.modules.source")}</th>
                <th>{t("admin.modules.last_synced")}</th>
                <th className="table-actions-col">{t("admin.modules.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((m) => (
                <tr key={m.module_key}>
                  <td>
                    <div style={{ fontWeight: 640 }}>{m.name}</div>
                    <div className="muted" style={{ fontSize: 12 }}>{m.module_key}</div>
                  </td>
                  <td>{m.category_path ? <span className="tag">{m.category_path}</span> : <span className="muted" style={{ fontSize: 12 }}>{t("admin.modules.unbound")}</span>}</td>
                  <td>
                    <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                      <span className="badge">{docTypeLabel(m.doc_type)}</span>
                      {m.mount ? <span className="badge">{mountLabel(m.mount, t)}</span> : null}
                    </div>
                  </td>
                  <td style={{ fontSize: 12 }}>
                    <div>
                      <span className="badge">{sourceTypeLabel(m.source_type, t)}</span>
                      {m.gitlab_branch ? <span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 3, marginLeft: 6 }}><GitBranch size={11} />{m.gitlab_branch}</span> : null}
                    </div>
                    {m.repo_url ? (
                      <div className="code-chip mt-1" style={{ maxWidth: 260, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{m.repo_url}</div>
                    ) : m.gitlab_path ? (
                      <div className="muted mt-1">{m.gitlab_path}</div>
                    ) : null}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {m.last_synced_at ? (
                      <>
                        {m.last_synced_commit ? <span className="code-chip">{m.last_synced_commit.slice(0, 8)}</span> : null}
                        <div className={m.last_synced_commit ? "mt-1" : ""}>{new Date(m.last_synced_at).toLocaleString()}</div>
                      </>
                    ) : (
                      <span>{t("admin.modules.never_synced")}</span>
                    )}
                  </td>
                  <td className="table-actions-cell">
                    <div className="row-actions">
                      <button className="icon-btn" onClick={() => openEdit(m)} aria-label={t("admin.categories.edit")}><Pencil size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
              {!pageItems.length ? (
                <tr><td colSpan={6}>
                  <EmptyState
                    icon={Boxes}
                    title={t("admin.modules.no_document_sources")}
                    hint={t("admin.modules.click_document_source_top_right_to_bind_a")}
                  />
                </td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
        <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPage={setPage} />
      </div>

      <Modal
        open={guideOpen}
        onClose={() => setGuideOpen(false)}
        title={t("admin.modules.integrationGuideTitle")}
        subtitle={t("admin.modules.integrationGuideSubtitle")}
        width={920}
        footer={<button className="button button-primary" onClick={() => setGuideOpen(false)}>{t("common.gotIt")}</button>}
      >
        <div className="docs-integration-guide">
          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">01</span>
              <div><h3>{t("admin.modules.integrationOverviewTitle")}</h3><p>{t("admin.modules.integrationOverviewCopy")}</p></div>
            </div>
            <ol className="docs-integration-steps">
              <li><span>1</span><div><strong>{t("admin.modules.integrationStepCreateTitle")}</strong><p>{t("admin.modules.integrationStepCreateCopy")}</p></div></li>
              <li><span>2</span><div><strong>{t("admin.modules.integrationStepTokenTitle")}</strong><p>{t("admin.modules.integrationStepTokenCopyPrefix")} <code>MODEX_DEPLOY_TOKEN</code>{t("admin.modules.integrationStepTokenCopySuffix")}</p></div></li>
              <li><span>3</span><div><strong>{t("admin.modules.integrationStepDeployTitle")}</strong><p>{t("admin.modules.integrationStepDeployCopyPrefix")} <code>docsctl deploy</code> {t("admin.modules.integrationStepDeployCopySuffix")}</p></div></li>
            </ol>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">02</span>
              <div><h3>{t("admin.modules.integrationFlowTitle")}</h3><p>{t("admin.modules.integrationFlowCopy")}</p></div>
            </div>
            <div className="docs-integration-diagram" aria-label={t("admin.modules.integrationFlowDiagramLabel")}>
              <Mermaid chart={INTEGRATION_FLOW} />
            </div>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">03</span>
              <div><h3>{t("admin.modules.basePathTitle")}</h3><p>{t("admin.modules.basePathCopy")}</p></div>
            </div>
            <div className="docs-integration-callout">
              <Info size={18} />
              <div>
                <strong>{t("admin.modules.basePathCalloutTitle")}</strong>
                <p>{t("admin.modules.basePathCalloutPrefix")} <code>DOCS_BASE</code>、<code>MODEX_DOCS_BASE</code>、<code>VITEPRESS_BASE</code> 和 <code>BASE_URL</code>{t("admin.modules.basePathCalloutSuffix")} <code>DOCS_BASE</code>。</p>
              </div>
            </div>
            <p className="docs-integration-note">{t("admin.modules.basePathNote")}</p>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">04</span>
              <div><h3>{t("admin.modules.frameworkExamplesTitle")}</h3><p>{t("admin.modules.frameworkExamplesCopyPrefix")} <code>"/"</code> {t("admin.modules.frameworkExamplesCopySuffix")}</p></div>
            </div>
            <div className="docs-base-config-grid">
              {BASE_CONFIGS.map((item) => (
                <article className="docs-base-config" key={item.name}>
                  <div className="docs-base-config__head"><strong>{item.name}</strong><code>{item.file}</code></div>
                  <pre>{item.code}</pre>
                  <CopyButton value={item.code} title={t("admin.modules.copyFrameworkConfig", { value1: item.name })} className="icon-btn docs-base-config__copy" />
                </article>
              ))}
            </div>
            <p className="docs-integration-note"><strong>Markdown / Static：</strong>{t("admin.modules.markdownStaticNote")}</p>
          </section>
        </div>
      </Modal>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={isEdit ? t("admin.modules.edit_document_source_value1", { value1: draft.name }) : t("admin.modules.connectDocSource")}
        subtitle={t("admin.modules.documentation_ownership_is_determined_by_the_selected_category")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{token && !isEdit ? t("admin.modules.done") : t("admin.categories.cancel")}</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim() || draft.category_ids.length === 0}>{isEdit ? t("admin.modules.save") : t("admin.categories.create")}</button>
          </>
        }
      >
        <div className="field">
          <label>{t("admin.categories.name")}</label>
          <input value={draft.name} placeholder={t("admin.modules.e_g_r_and_d_specification")} autoFocus onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          {isEdit ? <span className="field-hint">{t("admin.categories.id")} <span className="tree-keytag">{draft.module_key}</span></span> : <span className="field-hint">{t("admin.modules.id_module_key_is_auto_generated_by_the")}</span>}
        </div>
        <div className="field-row">
          <div className="field">
            <label>{t("admin.modules.documentation_framework")}</label>
            <Combobox options={DOC_TYPES} value={[draft.doc_type]} onChange={(v) => setDraft({ ...draft, doc_type: v[0] || "vitepress" })} multiple={false} placeholder={t("admin.modules.select_framework")} />
            <span className="field-hint">{draft.doc_type === "static" ? t("admin.modules.static_upload_an_existing_static_site_directory_directly") : compiled ? t("admin.modules.compiled_ci_builds_a_static_site_before_upload") : t("admin.modules.markdown_modex_renders_directly")}</span>
          </div>
          <div className="field">
            <label>{t("admin.modules.mount_method")}</label>
            <Combobox
              options={[{ value: "single", label: "single", hint: t("admin.modules.entire_site_as_one_document") }, { value: "split", label: "split", hint: t("admin.modules.split_top_level_directory_into_subdirectories") }]}
              value={[compiled ? "single" : draft.mount]}
              onChange={(v) => setDraft({ ...draft, mount: v[0] || "single" })}
              multiple={false}
            />
            <span className="field-hint">{compiled ? t("admin.modules.static_site_fixed_single") : t("admin.modules.split_is_available_only_for_markdown_based_sources")}</span>
          </div>
        </div>
        <div className="field">
          <label>{t("admin.modules.assigned_category")}</label>
          <Combobox options={categoryOptions} value={draft.category_ids} onChange={(category_ids) => setDraft({ ...draft, category_ids })} placeholder={t("admin.modules.bind_to_a_category")} />
          <span className="field-hint">{t("admin.modules.a_documentation_source_must_be_associated_with_at")}</span>
        </div>
        <div className="field">
          <label>{t("admin.categories.description")}</label>
          <input value={draft.description} placeholder={t("admin.modules.one_sentence_description_of_this_document_source")} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>

        {isEdit && draft.repo_url ? (
          <div className="field">
            <label>{t("admin.modules.source_repository_auto_filled_during_sync")}</label>
            <div style={{ display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
              <span className="badge">{draft.repo_type || "git"}</span>
              <span className="code-chip">{draft.repo_url}</span>
              {draft.gitlab_branch ? <span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 3, fontSize: 12 }}><GitBranch size={11} />{draft.gitlab_branch}</span> : null}
            </div>
          </div>
        ) : null}

        {token ? (
          <div className="field">
            <label>Deploy Token</label>
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <input
                readOnly
                value={maskToken(token.deploy_token)}
                style={{ fontFamily: "ui-monospace, monospace", fontSize: 12, letterSpacing: 1, flex: 1 }}
                tabIndex={-1}
              />
              <CopyButton value={token.deploy_token} title={t("admin.modules.copy_full_token")} />
              <button className="button" onClick={rotate}>{t("admin.modules.regenerate")}</button>
            </div>
            <span className="field-hint">{t("admin.modules.for_security_the_token_is_masked_click_the")} <code className="code-chip">MODEX_DEPLOY_TOKEN</code>{t("admin.modules.masked_repository_url_branch_is_automatically_populated_on")}</span>
          </div>
        ) : null}

        {token && draft.module_key ? (
          <div className="docs-source-quickstart">
            <div className="docs-source-quickstart__head">
              <div>
                <h3>{t("admin.modules.push_documentation")}</h3>
                <p>{t("admin.modules.use_a_single_command_for_local_debugging_for")}</p>
              </div>
            </div>
            <div className="docs-source-code">
              <div className="docs-source-code__head">
                <span>{t("admin.modules.local_one_time_deployment")}</span>
                <CopyButton value={localDeployCommand(token.deploy_url)} title={t("admin.modules.copy_on_premises_deployment_command")} label={t("component.ui.copyButton.copy")} className="button" />
              </div>
              <pre>{localDeployCommand(token.deploy_url)}</pre>
            </div>
            <div className="docs-source-code">
              <div className="docs-source-code__head">
                <span>GitLab CI</span>
                <CopyButton value={gitlabSnippet(token.deploy_url, templateInclude)} title={t("admin.modules.copy_gitlab_ci_configuration")} label={t("component.ui.copyButton.copy")} className="button" />
              </div>
              <pre>{gitlabSnippet(token.deploy_url, templateInclude)}</pre>
            </div>
          </div>
        ) : null}
      </Modal>
    </AdminShell>
  );
}

function docTypeLabel(value?: string) {
  if (!value) return "-";
  const labels: Record<string, string> = {
    vitepress: "VitePress",
    vuepress: "VuePress",
    fumadocs: "Fumadocs",
    docusaurus: "Docusaurus",
    mkdocs: "MkDocs",
    honkit: "HonKit",
    markdown: "Markdown",
    static: "Static"
  };
  return labels[value] || value;
}

function mountLabel(value: string, t: (key: string) => string) {
  if (value === "single") return t("admin.modules.mount_single");
  if (value === "split") return t("admin.modules.mount_split");
  return value;
}

function sourceTypeLabel(value: string | undefined, t: (key: string) => string) {
  if (value === "gitlab") return "GitLab";
  if (value === "github") return "GitHub";
  if (value === "manual") return t("admin.modules.manual_source");
  return t("admin.modules.manual_source");
}
