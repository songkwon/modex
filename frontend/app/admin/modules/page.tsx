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
    User->>CI: 配置 Token、module key 和构建参数
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

function localDeployCommand(moduleKey: string, deployUrl?: string) {
  return [
    "docsctl deploy \\",
    "  --source /path/to/docs \\",
    `  --module ${moduleKey || "<module_key>"} \\`,
    "  --version latest \\",
    `  --deploy-url ${deployUrl || "https://modex.example.com/api/deploy"} \\`,
    "  --token $MODEX_DEPLOY_TOKEN",
  ].join("\n");
}

function gitlabSnippet(moduleKey: string, deployUrl?: string, builder = "vitepress") {
  const output = builder === "vuepress"
    ? "docs/.vuepress/dist"
    : builder === "fumadocs"
      ? "out"
      : builder === "docusaurus"
        ? "build"
        : builder === "mkdocs"
          ? "site"
          : builder === "honkit" || builder === "gitbook"
            ? "_book"
            : builder === "static"
              ? "dist"
              : "docs/.vitepress/dist";
  const build = builder === "static" || builder === "markdown" ? "" : builder === "mkdocs" ? "mkdocs build" : "npm ci && npm run docs:build";
  return `include:
  - remote: "https://raw.githubusercontent.com/songkwon/modex/main/deploy/ci/modex-docs.gitlab-ci.yml"

variables:
  MODEX_MODULE_KEY: "${moduleKey || "<module_key>"}"
  DOCS_BUILDER: "${builder}"
${build ? `  DOCS_BUILD: "${build}"\n` : ""}  DOCS_OUTPUT: "${output}"
  MODEX_DEPLOY_URL: "${deployUrl || "https://modex.example.com/api/deploy"}"

# 在 GitLab Settings > CI/CD > Variables 中添加：
# MODEX_DEPLOY_TOKEN = 从 Modex 文档源复制的 Deploy Token（Masked + Protected）`;
}

export default function AdminModulesPage() {
  const { t } = useI18n();
  const DOC_TYPES = [
    { value: "vitepress", label: "VitePress", hint: t("legacy.b1f65b3ea73b") },
    { value: "vuepress", label: "VuePress", hint: t("legacy.3c802754e650") },
    { value: "fumadocs", label: "Fumadocs", hint: t("legacy.b8f6b2bb0ab9") },
    { value: "docusaurus", label: "Docusaurus", hint: t("legacy.b1f65b3ea73b") },
    { value: "mkdocs", label: "MkDocs", hint: t("legacy.068e0173a73e") },
    { value: "honkit", label: "HonKit / GitBook", hint: t("legacy.b1f65b3ea73b") },
    { value: "markdown", label: "Markdown", hint: t("legacy.d50ad8bed8a9") },
    { value: "static", label: "Static", hint: t("legacy.50296fe0c571") },
  ];
  const [categories, setCategories] = useState<Category[]>([]);
  const [keyword, setKeyword] = useState("");
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [guideOpen, setGuideOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [token, setToken] = useState<{ deploy_token: string; deploy_url: string } | null>(null);
  const isEdit = !!draft.module_key;

  const { items: pageItems, total, page, setPage, error: loadError, reload } = usePaged<ModuleInfo>(
    "/api/admin/modules",
    PAGE_SIZE,
    keyword.trim(),
  );

  useEffect(() => {
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
    if (!confirm(t("legacy.076fd4d980e9"))) return;
    try {
      setToken(await rotateDeployToken(draft.module_key));
    } catch (e) {
      setError(String(e));
    }
  }

  const compiled = COMPILED.has(draft.doc_type);

  return (
    <AdminShell title={t("legacy.608c2486d555")} kicker="Doc Sources" description={t("legacy.c3c46c5a26a5")}>
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input placeholder={t("legacy.39516b942ec6")} value={keyword} onChange={(e) => setKeyword(e.target.value)} />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button" onClick={() => setGuideOpen(true)}><BookOpen size={16} /> {t("legacy.6620ed97d068")}</button>
          <button className="button button-primary" onClick={openCreate}><Plus size={16} /> {t("legacy.c79d024b319c")}</button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.79963951e270")}</th>
                <th>{t("legacy.515559957fd3")}</th>
                <th>{t("legacy.8c25f482693a")}</th>
                <th>{t("legacy.89b2ecbe1f09")}</th>
                <th>{t("legacy.a056164d4abc")}</th>
                <th style={{ textAlign: "right" }}>{t("legacy.ed31fbb483ee")}</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((m) => (
                <tr key={m.module_key}>
                  <td>
                    <div style={{ fontWeight: 640 }}>{m.name}</div>
                    <div className="muted" style={{ fontSize: 12 }}>{m.module_key}</div>
                  </td>
                  <td>{m.category_path ? <span className="tag">{m.category_path}</span> : <span className="muted" style={{ fontSize: 12 }}>{t("legacy.e026c6693dc5")}</span>}</td>
                  <td>
                    <span className="badge">{m.doc_type || "—"}</span>
                    {m.mount ? <span className="badge" style={{ marginLeft: 4 }}>{m.mount}</span> : null}
                  </td>
                  <td style={{ fontSize: 12 }}>
                    {m.repo_url ? (
                      <span className="code-chip" style={{ maxWidth: 220, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", display: "inline-block" }}>{m.repo_url}</span>
                    ) : <span className="muted">{t("legacy.9a964fbc3d5a")}</span>}
                    {m.gitlab_branch ? <span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 3, marginLeft: 6 }}><GitBranch size={11} />{m.gitlab_branch}</span> : null}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {m.last_synced_commit ? m.last_synced_commit.slice(0, 8) : "—"}
                    {m.last_synced_at ? <div>{new Date(m.last_synced_at).toLocaleString()}</div> : null}
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="icon-btn" onClick={() => openEdit(m)} aria-label={t("legacy.051836569928")}><Pencil size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
              {!pageItems.length ? (
                <tr><td colSpan={6}>
                  <EmptyState
                    icon={Boxes}
                    title={t("legacy.9392e8487579")}
                    hint={t("legacy.bd40a1991073")}
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
        title={t("legacy.6ae3d74ff5e4")}
        subtitle={t("legacy.2a7d9ad2ad5e")}
        width={920}
        footer={<button className="button button-primary" onClick={() => setGuideOpen(false)}>{t("legacy.de32e20193ad")}</button>}
      >
        <div className="docs-integration-guide">
          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">01</span>
              <div><h3>{t("legacy.893814adfdf1")}</h3><p>{t("legacy.2b1ffb1325c8")}</p></div>
            </div>
            <ol className="docs-integration-steps">
              <li><span>1</span><div><strong>{t("legacy.17cd44e62255")}</strong><p>{t("legacy.290191fa3517")}</p></div></li>
              <li><span>2</span><div><strong>{t("legacy.6d5c388675ec")}</strong><p>{t("legacy.ab65dbc33824")} <code>MODEX_DEPLOY_TOKEN</code>{t("legacy.c2aca1afd3d4")}</p></div></li>
              <li><span>3</span><div><strong>{t("legacy.77eca0421917")}</strong><p>{t("legacy.3e62c950eb76")} <code>docsctl deploy</code> {t("legacy.efb128557f40")}</p></div></li>
            </ol>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">02</span>
              <div><h3>{t("legacy.8d0c97460762")}</h3><p>{t("legacy.d45226595dff")}</p></div>
            </div>
            <div className="docs-integration-diagram" aria-label={t("legacy.aee26e54d85e")}>
              <Mermaid chart={INTEGRATION_FLOW} />
            </div>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">03</span>
              <div><h3>{t("legacy.d5bbc1fddba2")}</h3><p>{t("legacy.03791d868366")}</p></div>
            </div>
            <div className="docs-integration-callout">
              <Info size={18} />
              <div>
                <strong>{t("legacy.f713369f5d88")}</strong>
                <p>{t("legacy.60fa8bef54c4")} <code>DOCS_BASE</code>、<code>MODEX_DOCS_BASE</code>、<code>VITEPRESS_BASE</code> 和 <code>BASE_URL</code>{t("legacy.0a09fd50358f")} <code>DOCS_BASE</code>。</p>
              </div>
            </div>
            <p className="docs-integration-note">{t("legacy.b6a6874227ef")}</p>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">04</span>
              <div><h3>{t("legacy.a7a531a8e54b")}</h3><p>{t("legacy.670ec25af841")} <code>"/"</code> {t("legacy.b167f4e24da9")}</p></div>
            </div>
            <div className="docs-base-config-grid">
              {BASE_CONFIGS.map((item) => (
                <article className="docs-base-config" key={item.name}>
                  <div className="docs-base-config__head"><strong>{item.name}</strong><code>{item.file}</code></div>
                  <pre>{item.code}</pre>
                  <CopyButton value={item.code} title={t("legacy.5841e7450ccd", { value1: item.name })} className="icon-btn docs-base-config__copy" />
                </article>
              ))}
            </div>
            <p className="docs-integration-note"><strong>Markdown / Static：</strong>{t("legacy.59d2ee01df1f")}</p>
          </section>
        </div>
      </Modal>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={isEdit ? t("legacy.7f01a8cff581", { value1: draft.name }) : t("legacy.c79d024b319c")}
        subtitle={t("legacy.6116c4e10bd9")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{token && !isEdit ? t("legacy.c0b3fbff51cc") : t("legacy.2cd0f3be8738")}</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim() || draft.category_ids.length === 0}>{isEdit ? t("legacy.a3030bf8f16d") : t("legacy.cde2cd071d25")}</button>
          </>
        }
      >
        <div className="field">
          <label>{t("legacy.909712f2847d")}</label>
          <input value={draft.name} placeholder={t("legacy.c3a2772b2723")} autoFocus onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          {isEdit ? <span className="field-hint">{t("legacy.301c9461c1d0")} <span className="tree-keytag">{draft.module_key}</span></span> : <span className="field-hint">{t("legacy.c1d3d4d56a2d")}</span>}
        </div>
        <div className="field-row">
          <div className="field">
            <label>{t("legacy.1fbf269e8a39")}</label>
            <Combobox options={DOC_TYPES} value={[draft.doc_type]} onChange={(v) => setDraft({ ...draft, doc_type: v[0] || "vitepress" })} multiple={false} placeholder={t("legacy.7a6a8975d3d5")} />
            <span className="field-hint">{draft.doc_type === "static" ? t("legacy.f5641a61d3c0") : compiled ? t("legacy.0b2128fdeb1d") : t("legacy.3d96681588a1")}</span>
          </div>
          <div className="field">
            <label>{t("legacy.2f5b025f0850")}</label>
            <Combobox
              options={[{ value: "single", label: "single", hint: t("legacy.d8d532298db1") }, { value: "split", label: "split", hint: t("legacy.faa4e4d04f89") }]}
              value={[compiled ? "single" : draft.mount]}
              onChange={(v) => setDraft({ ...draft, mount: v[0] || "single" })}
              multiple={false}
            />
            <span className="field-hint">{compiled ? t("legacy.0a45ff65dd41") : t("legacy.3c36701bd788")}</span>
          </div>
        </div>
        <div className="field">
          <label>{t("legacy.411f6ff38732")}</label>
          <Combobox options={categoryOptions} value={draft.category_ids} onChange={(category_ids) => setDraft({ ...draft, category_ids })} placeholder={t("legacy.b110db08e6eb")} />
          <span className="field-hint">{t("legacy.c54587ad05db")}</span>
        </div>
        <div className="field">
          <label>{t("legacy.dc2ba467fc7a")}</label>
          <input value={draft.description} placeholder={t("legacy.48864062f792")} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>

        {isEdit && draft.repo_url ? (
          <div className="field">
            <label>{t("legacy.d082c4376a60")}</label>
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
              <CopyButton value={token.deploy_token} title={t("legacy.012172294c04")} />
              <button className="button" onClick={rotate}>{t("legacy.3221a042ea36")}</button>
            </div>
            <span className="field-hint">{t("legacy.75350b2f827b")} <code className="code-chip">MODEX_DEPLOY_TOKEN</code>{t("legacy.c3986a9c2b62")}</span>
          </div>
        ) : null}

        {token && draft.module_key ? (
          <div className="docs-source-quickstart">
            <div className="docs-source-quickstart__head">
              <div>
                <h3>{t("legacy.17d88e668229")}</h3>
                <p>{t("legacy.e3e942324ff0")}</p>
              </div>
            </div>
            <div className="docs-source-code">
              <div className="docs-source-code__head">
                <span>{t("legacy.e1d1eef4e7ef")}</span>
                <CopyButton value={localDeployCommand(draft.module_key, token.deploy_url)} title={t("legacy.e96684841a07")} label={t("legacy.63d90d977348")} className="button" />
              </div>
              <pre>{localDeployCommand(draft.module_key, token.deploy_url)}</pre>
            </div>
            <div className="docs-source-code">
              <div className="docs-source-code__head">
                <span>GitLab CI</span>
                <CopyButton value={gitlabSnippet(draft.module_key, token.deploy_url, draft.doc_type)} title={t("legacy.fa9e0153ca47")} label={t("legacy.63d90d977348")} className="button" />
              </div>
              <pre>{gitlabSnippet(draft.module_key, token.deploy_url, draft.doc_type)}</pre>
            </div>
          </div>
        ) : null}
      </Modal>
    </AdminShell>
  );
}
