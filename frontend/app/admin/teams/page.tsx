"use client";

import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Search, Trash2, UserPlus, UsersRound } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { UserSelect } from "@/components/ui/user-select";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";

const PAGE_SIZE = 8;
import {
  createTeam,
  updateTeam,
  deleteTeam,
  getCategories,
} from "@/lib/api";
import type { Team, Category } from "@/types/modex";
import { usePaged } from "@/lib/use-paged";

function flattenCategories(cats: Category[] | null | undefined): Category[] {
  if (!cats) return [];
  const out: Category[] = [];
  const walk = (nodes: Category[]) => {
    for (const n of nodes) {
      out.push(n);
      if (n.children) walk(n.children);
    }
  };
  walk(cats);
  return out;
}

type Draft = {
  key: string;
  name: string;
  description: string;
  leaders: string[];
  members: string[];
};

const emptyDraft: Draft = { key: "", name: "", description: "", leaders: [], members: [] };

export default function AdminTeamsPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [isEdit, setIsEdit] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [picker, setPicker] = useState<null | "leader" | "members">(null);

  const { items: pageTeams, total, page, setPage, error: loadError, reload } = usePaged<Team>(
    "/api/admin/teams",
    PAGE_SIZE,
    keyword.trim(),
  );

  // Categories are needed in full to derive each team's owned domains.
  useEffect(() => {
    getCategories().then((tree) => setCategories(tree || [])).catch(() => {});
  }, []);

  const flatCats = useMemo(() => flattenCategories(categories), [categories]);
  function ownedDomainsFor(teamKey: string): string[] {
    return flatCats.filter((c) => c.responsible_team === teamKey).map((c) => c.name);
  }

  function openCreate() {
    setDraft(emptyDraft);
    setIsEdit(false);
    setModalOpen(true);
  }
  function openEdit(t: Team) {
    setDraft({ key: t.key, name: t.name || "", description: t.description || "", leaders: t.leaders || [], members: t.members || [] });
    setIsEdit(true);
    setModalOpen(true);
  }

  async function submit() {
    setError("");
    try {
      if (isEdit) {
        await updateTeam(draft.key, { name: draft.name, description: draft.description, leaders: draft.leaders, members: draft.members });
      } else {
        await createTeam({
          key: draft.key,
          name: draft.name || undefined,
          description: draft.description || undefined,
          leaders: draft.leaders.length ? draft.leaders : undefined,
          members: draft.members.length ? draft.members : undefined,
        });
      }
      setModalOpen(false);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  async function removeTeam(t: Team) {
    if (!confirm(`确认删除团队 ${t.key}? 关联的分类负责方不会自动清除。`)) return;
    try {
      await deleteTeam(t.key);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title="团队管理"
      kicker="Teams"
      description="文档维护团队（负责人 + 成员）。负责人可直接添加成员。团队可被指定为某个分类的负责方，负责维护该分类下的文档结构与归属。"
    >
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder="搜索团队名 / 标识 / 负责人 / 成员"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> 新增团队
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>团队</th>
                <th>负责人</th>
                <th>成员</th>
                <th>负责分类</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {pageTeams.map((t) => {
                const owned = ownedDomainsFor(t.key);
                return (
                  <tr key={t.key}>
                    <td>
                      <div style={{ fontWeight: 640 }}>{t.name || t.key}</div>
                      <div className="muted" style={{ fontSize: 12 }}>{t.key}{t.description ? " · " + t.description : ""}</div>
                    </td>
                    <td>
                      {(t.leaders || []).length > 0 ? (
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                          {(t.leaders || []).map((l) => <span key={l} className="badge badge-success">{l}</span>)}
                        </div>
                      ) : (
                        <span className="muted">—</span>
                      )}
                    </td>
                    <td>
                      {(t.members || []).length > 0 ? (
                        <span className="tag">{(t.members || []).length} 人</span>
                      ) : (
                        <span className="muted" style={{ fontSize: 12 }}>—</span>
                      )}
                    </td>
                    <td>
                      {owned.length > 0 ? owned.map((n) => <span key={n} className="tag" style={{ marginRight: 4 }}>{n}</span>) : <span className="muted" style={{ fontSize: 12 }}>未绑定分类</span>}
                    </td>
                    <td>
                      <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                        <button className="icon-btn" onClick={() => openEdit(t)} aria-label="编辑"><Pencil size={14} /></button>
                        <button className="icon-btn danger" onClick={() => removeTeam(t)} aria-label="删除"><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {total === 0 ? (
                <tr><td colSpan={5}>
                  <EmptyState
                    icon={UsersRound}
                    title={keyword ? "没有匹配的团队" : "暂无团队"}
                    hint={keyword ? "换个关键词试试。" : "创建一个团队，并将其设为某个分类的负责方，即可开始维护文档结构。"}
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
        title={isEdit ? `编辑团队 · ${draft.key}` : "新增团队"}
        subtitle={isEdit ? "更新团队信息与成员" : "负责人会自动加入成员列表；标识可用于分类负责方绑定"}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>取消</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim() || draft.leaders.length === 0}>
              {isEdit ? "保存" : "创建团队"}
            </button>
          </>
        }
      >
        <div className="field">
          <label>名称 *</label>
          <input value={draft.name} placeholder="如 研发规范组" autoFocus onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          {isEdit ? <span className="field-hint">标识 <code className="code-chip">{draft.key}</code>（系统生成，不可修改）</span> : <span className="field-hint">标识由系统自动生成。</span>}
        </div>
        <div className="field">
          <label>描述</label>
          <input value={draft.description} placeholder="一句话说明团队职责" onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>
        <div className="field">
          <label>负责人 *</label>
          <div className="picker-field">
            {draft.leaders.length > 0 ? (
              draft.leaders.map((l) => <span key={l} className="tag">{l}</span>)
            ) : (
              <span className="muted" style={{ fontSize: 13 }}>未指定</span>
            )}
            <button type="button" className="button" onClick={() => setPicker("leader")}>
              <UserPlus size={14} /> 选择负责人
            </button>
          </div>
          <span className="field-hint">负责人至少 1 人（可多人），可管理本团队负责分类下的内容；保存后会自动加入成员。</span>
        </div>
        <div className="field">
          <label>成员</label>
          <div className="picker-field">
            {draft.members.length > 0 ? (
              draft.members.map((m) => <span key={m} className="tag">{m}</span>)
            ) : (
              <span className="muted" style={{ fontSize: 13 }}>暂无成员</span>
            )}
            <button type="button" className="button" onClick={() => setPicker("members")}>
              <UserPlus size={14} /> 添加成员
            </button>
          </div>
        </div>
      </Modal>

      <Modal
        open={picker !== null}
        onClose={() => setPicker(null)}
        title={picker === "leader" ? "选择负责人" : "添加成员"}
        subtitle={picker === "leader" ? "搜索并按部门展开多选（至少 1 人）" : "搜索并按部门展开多选"}
        footer={<button className="button button-primary" onClick={() => setPicker(null)}>完成</button>}
      >
        {picker === "leader" ? (
          <UserSelect value={draft.leaders} onChange={(leaders) => setDraft({ ...draft, leaders })} />
        ) : picker === "members" ? (
          <UserSelect value={draft.members} onChange={(members) => setDraft({ ...draft, members })} />
        ) : null}
      </Modal>
    </AdminShell>
  );
}
