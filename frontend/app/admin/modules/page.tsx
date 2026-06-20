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

const PAGE_SIZE = 8;
const DOC_TYPES = [
  { value: "vitepress", label: "VitePress", hint: "编译型 · 静态站" },
  { value: "vuepress", label: "VuePress", hint: "编译型" },
  { value: "fumadocs", label: "Fumadocs", hint: "编译型 · Next/MDX" },
  { value: "docusaurus", label: "Docusaurus", hint: "编译型 · 静态站" },
  { value: "mkdocs", label: "MkDocs", hint: "编译型 · Python" },
  { value: "honkit", label: "HonKit / GitBook", hint: "编译型 · 静态站" },
  { value: "markdown", label: "Markdown", hint: "modex 渲染" },
  { value: "static", label: "Static", hint: "通用静态站" },
];
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
    if (!confirm("重新生成 Deploy Token？旧 Token 将立即失效。")) return;
    try {
      setToken(await rotateDeployToken(draft.module_key));
    } catch (e) {
      setError(String(e));
    }
  }

  const compiled = COMPILED.has(draft.doc_type);

  return (
    <AdminShell title="文档源管理" kicker="Doc Sources" description="接入 Git / SVN 文档仓库：选择文档框架、绑定分类、生成 Deploy Token，CI 推送后即归档到对应分类。">
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input placeholder="搜索名称 / key / 仓库" value={keyword} onChange={(e) => setKeyword(e.target.value)} />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button" onClick={() => setGuideOpen(true)}><BookOpen size={16} /> 接入说明</button>
          <button className="button button-primary" onClick={openCreate}><Plus size={16} /> 接入文档源</button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>文档源</th>
                <th>分类</th>
                <th>框架 / 挂载</th>
                <th>仓库</th>
                <th>最后同步</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((m) => (
                <tr key={m.module_key}>
                  <td>
                    <div style={{ fontWeight: 640 }}>{m.name}</div>
                    <div className="muted" style={{ fontSize: 12 }}>{m.module_key}</div>
                  </td>
                  <td>{m.category_path ? <span className="tag">{m.category_path}</span> : <span className="muted" style={{ fontSize: 12 }}>未绑定</span>}</td>
                  <td>
                    <span className="badge">{m.doc_type || "—"}</span>
                    {m.mount ? <span className="badge" style={{ marginLeft: 4 }}>{m.mount}</span> : null}
                  </td>
                  <td style={{ fontSize: 12 }}>
                    {m.repo_url ? (
                      <span className="code-chip" style={{ maxWidth: 220, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", display: "inline-block" }}>{m.repo_url}</span>
                    ) : <span className="muted">未同步</span>}
                    {m.gitlab_branch ? <span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 3, marginLeft: 6 }}><GitBranch size={11} />{m.gitlab_branch}</span> : null}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {m.last_synced_commit ? m.last_synced_commit.slice(0, 8) : "—"}
                    {m.last_synced_at ? <div>{new Date(m.last_synced_at).toLocaleString()}</div> : null}
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="icon-btn" onClick={() => openEdit(m)} aria-label="编辑"><Pencil size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
              {!pageItems.length ? (
                <tr><td colSpan={6}>
                  <EmptyState
                    icon={Boxes}
                    title="暂无文档源"
                    hint="点击右上角「接入文档源」绑定一个文档仓库到分类，CI 推送后即可归档。"
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
        title="文档源接入说明"
        subtitle="从创建文档源到 CI 构建、打包和发布"
        width={920}
        footer={<button className="button button-primary" onClick={() => setGuideOpen(false)}>知道了</button>}
      >
        <div className="docs-integration-guide">
          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">01</span>
              <div><h3>接入流程</h3><p>生产环境推荐由文档仓库的 GitLab CI 自动发布。</p></div>
            </div>
            <ol className="docs-integration-steps">
              <li><span>1</span><div><strong>创建文档源</strong><p>选择框架和归属分类，创建后复制 Deploy Token。</p></div></li>
              <li><span>2</span><div><strong>配置 CI 变量</strong><p>将 Token 保存为 <code>MODEX_DEPLOY_TOKEN</code>，并设置 module key、构建器和输出目录。</p></div></li>
              <li><span>3</span><div><strong>引入 CI 模板</strong><p>推送默认分支后，模板会执行 <code>docsctl deploy</code> 完成校验、构建、打包和上传。</p></div></li>
            </ol>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">02</span>
              <div><h3>构建与发布链路</h3><p>Deploy Token 只放在 CI Secret 中，不要提交到文档仓库。</p></div>
            </div>
            <div className="docs-integration-diagram" aria-label="文档接入 UML 时序图">
              <Mermaid chart={INTEGRATION_FLOW} />
            </div>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">03</span>
              <div><h3>打包型文档如何处理 base</h3><p>VitePress、VuePress 等静态站需要在构建时知道最终挂载路径。</p></div>
            </div>
            <div className="docs-integration-callout">
              <Info size={18} />
              <div>
                <strong>docsctl 会自动注入，无需在 CI 中手工拼接路径</strong>
                <p>运行构建命令时会同时写入 <code>DOCS_BASE</code>、<code>MODEX_DOCS_BASE</code>、<code>VITEPRESS_BASE</code> 和 <code>BASE_URL</code>。文档项目的配置文件仍需读取其中一个变量，推荐统一读取 <code>DOCS_BASE</code>。</p>
              </div>
            </div>
            <p className="docs-integration-note">构建完成后，docsctl 还会修正常见 HTML、CSS 和 manifest 中的根路径资源引用。这是兼容兜底；涉及客户端路由、运行时代码或框架生成链接时，仍应在框架配置中正确使用 base。</p>
          </section>

          <section className="docs-integration-section">
            <div className="docs-integration-heading">
              <span className="docs-integration-index">04</span>
              <div><h3>配置文件修改示例</h3><p>保留 <code>"/"</code> 作为本地独立运行时的默认值。</p></div>
            </div>
            <div className="docs-base-config-grid">
              {BASE_CONFIGS.map((item) => (
                <article className="docs-base-config" key={item.name}>
                  <div className="docs-base-config__head"><strong>{item.name}</strong><code>{item.file}</code></div>
                  <pre>{item.code}</pre>
                  <CopyButton value={item.code} title={`复制 ${item.name} 配置`} className="icon-btn docs-base-config__copy" />
                </article>
              ))}
            </div>
            <p className="docs-integration-note"><strong>Markdown / Static：</strong>Markdown 资源由 docsctl 打包时处理；已有静态目录可直接使用 Static，并由产物重写机制兼容常见绝对资源路径。HonKit / GitBook 若无可用的 base 配置，也使用该兜底机制。</p>
          </section>
        </div>
      </Modal>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={isEdit ? `编辑文档源 · ${draft.name}` : "接入文档源"}
        subtitle="文档归属由所选分类决定；Deploy Token 用于 CI 推送鉴权"
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{token && !isEdit ? "完成" : "取消"}</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim() || draft.category_ids.length === 0}>{isEdit ? "保存" : "创建"}</button>
          </>
        }
      >
        <div className="field">
          <label>名称 *</label>
          <input value={draft.name} placeholder="如 研发规范" autoFocus onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          {isEdit ? <span className="field-hint">标识 <span className="tree-keytag">{draft.module_key}</span></span> : <span className="field-hint">标识（module key）由系统自动生成。</span>}
        </div>
        <div className="field-row">
          <div className="field">
            <label>文档框架</label>
            <Combobox options={DOC_TYPES} value={[draft.doc_type]} onChange={(v) => setDraft({ ...draft, doc_type: v[0] || "vitepress" })} multiple={false} placeholder="选择框架…" />
            <span className="field-hint">{draft.doc_type === "static" ? "Static：直接上传已有静态站目录。" : compiled ? "编译型：CI 构建为静态站后上传。" : "Markdown：modex 直接渲染。"}</span>
          </div>
          <div className="field">
            <label>挂载方式</label>
            <Combobox
              options={[{ value: "single", label: "single", hint: "整站一篇" }, { value: "split", label: "split", hint: "顶层目录拆子级" }]}
              value={[compiled ? "single" : draft.mount]}
              onChange={(v) => setDraft({ ...draft, mount: v[0] || "single" })}
              multiple={false}
            />
            <span className="field-hint">{compiled ? "静态站固定 single。" : "split 仅 markdown 型可用。"}</span>
          </div>
        </div>
        <div className="field">
          <label>归属分类</label>
          <Combobox options={categoryOptions} value={draft.category_ids} onChange={(category_ids) => setDraft({ ...draft, category_ids })} placeholder="绑定到一个分类…" />
          <span className="field-hint">文档源必须关联至少一个分类；只能选择你所在团队负责的分类。</span>
        </div>
        <div className="field">
          <label>描述</label>
          <input value={draft.description} placeholder="一句话说明这个文档源" onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>

        {isEdit && draft.repo_url ? (
          <div className="field">
            <label>来源仓库（同步时自动填充）</label>
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
              <CopyButton value={token.deploy_token} title="复制完整 Token" />
              <button className="button" onClick={rotate}>重新生成</button>
            </div>
            <span className="field-hint">出于安全考虑 Token 仅以掩码展示，点击复制按钮可拷贝完整值。在文档仓库 GitLab CI 变量里设为 <code className="code-chip">MODEX_DEPLOY_TOKEN</code>（Masked）。仓库地址/分支会在首次 CI 推送时自动带过来。</span>
          </div>
        ) : null}

        {token && draft.module_key ? (
          <div className="docs-source-quickstart">
            <div className="docs-source-quickstart__head">
              <div>
                <h3>推送文档</h3>
                <p>本地调试用一条命令；生产环境推荐 GitLab CI 自动推送。</p>
              </div>
            </div>
            <div className="docs-source-code">
              <div className="docs-source-code__head">
                <span>本地一次性部署</span>
                <CopyButton value={localDeployCommand(draft.module_key, token.deploy_url)} title="复制本地部署命令" label="复制" className="button" />
              </div>
              <pre>{localDeployCommand(draft.module_key, token.deploy_url)}</pre>
            </div>
            <div className="docs-source-code">
              <div className="docs-source-code__head">
                <span>GitLab CI</span>
                <CopyButton value={gitlabSnippet(draft.module_key, token.deploy_url, draft.doc_type)} title="复制 GitLab CI 配置" label="复制" className="button" />
              </div>
              <pre>{gitlabSnippet(draft.module_key, token.deploy_url, draft.doc_type)}</pre>
            </div>
          </div>
        ) : null}
      </Modal>
    </AdminShell>
  );
}
