"use client";

import { useEffect, useMemo, useState } from "react";
import { Boxes, GitBranch, Pencil, Plus, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { CopyButton } from "@/components/ui/copy-button";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
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
          <button className="button button-primary" onClick={openCreate}><Plus size={16} /> 接入文档源</button>
        </div>
      </div>

      <section className="docs-source-guide">
        <div>
          <p className="docs-source-guide__eyebrow">推荐接入方式</p>
          <h2>先创建文档源，再让 GitLab CI 自动推送</h2>
          <p>
            docsctl 会在文档仓库里完成构建、打包和上传。用户只需要在 Modex 创建文档源、复制 Deploy Token，
            然后把 GitLab CI 模板 include 到文档仓库即可。
          </p>
        </div>
        <ol>
          <li>点击「接入文档源」，选择框架和归属分类。</li>
          <li>创建后复制 Deploy Token，保存为 GitLab CI 变量 <code>MODEX_DEPLOY_TOKEN</code>。</li>
          <li>在文档仓库加入 CI 模板，设置 <code>MODEX_MODULE_KEY</code>、<code>DOCS_BUILDER</code>、<code>DOCS_OUTPUT</code>。</li>
          <li>推送默认分支，CI 会自动执行 <code>docsctl deploy</code>。</li>
        </ol>
        <div className="docs-source-guide__note">
          支持 VitePress、VuePress、Fumadocs、Docusaurus、MkDocs、HonKit/GitBook、Markdown；其他 HTML 目录可用 Static 接入。docsctl 会自动注入文档 base，并修正常见的根路径静态资源引用。
        </div>
      </section>

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
