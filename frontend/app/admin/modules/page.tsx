"use client";

import { useEffect, useMemo, useState } from "react";
import { Boxes, GitBranch, Pencil, Plus, Search } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { CopyButton } from "@/components/ui/copy-button";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import { createModule, getCategories, getDeployToken, rotateDeployToken, updateModule } from "@/lib/api";
import { usePaged } from "@/lib/use-paged";
import type { Category, ModuleInfo } from "@/types/modex";

const PAGE_SIZE = 8;
const DOC_TYPES = [
  { value: "vitepress", label: "VitePress（编译）" },
  { value: "vuepress", label: "VuePress（编译）" },
  { value: "fumadocs", label: "Fumadocs（编译）" },
  { value: "markdown", label: "Markdown（modex 渲染）" },
];
const COMPILED = new Set(["vitepress", "vuepress", "fumadocs"]);

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
    getCategories().then((tree) => setCategories(tree || [])).catch(() => {});
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
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim()}>{isEdit ? "保存" : "创建"}</button>
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
          </div>
          <div className="field">
            <label>挂载方式</label>
            <Combobox
              options={[{ value: "single", label: "single（整站一篇）" }, { value: "split", label: "split（顶层目录拆子级）" }]}
              value={[compiled ? "single" : draft.mount]}
              onChange={(v) => setDraft({ ...draft, mount: v[0] || "single" })}
              multiple={false}
            />
            <span className="field-hint">{compiled ? "编译型固定 single。" : "split 仅 markdown 型可用。"}</span>
          </div>
        </div>
        <div className="field">
          <label>归属分类</label>
          <Combobox options={categoryOptions} value={draft.category_ids} onChange={(category_ids) => setDraft({ ...draft, category_ids })} placeholder="绑定到一个分类…" />
          <span className="field-hint">文档将挂在所选分类下（一仓一锚点）。</span>
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
      </Modal>
    </AdminShell>
  );
}
