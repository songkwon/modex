"use client";

import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Trash2, UserPlus, X } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Combobox } from "@/components/ui/combobox";
import {
  getTeams,
  createTeam,
  updateTeam,
  deleteTeam,
  addTeamMember,
  removeTeamMember,
  getCategories,
} from "@/lib/api";
import type { Team, Category } from "@/types/modex";

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
  leader: string;
  members: string[];
};

const emptyDraft: Draft = { key: "", name: "", description: "", leader: "", members: [] };

export default function AdminTeamsPage() {
  const [teams, setTeams] = useState<Team[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [isEdit, setIsEdit] = useState(false);
  const [addMemberFor, setAddMemberFor] = useState<Record<string, string>>({});

  async function refresh() {
    try {
      const [ts, tree] = await Promise.all([getTeams(), getCategories()]);
      setTeams(ts || []);
      setCategories(tree || []);
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }

  useEffect(() => {
    refresh();
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
    setDraft({ key: t.key, name: t.name || "", description: t.description || "", leader: t.leader || "", members: t.members || [] });
    setIsEdit(true);
    setModalOpen(true);
  }

  async function submit() {
    setError("");
    try {
      if (isEdit) {
        await updateTeam(draft.key, { name: draft.name, description: draft.description, leader: draft.leader, members: draft.members });
      } else {
        await createTeam({
          key: draft.key,
          name: draft.name || undefined,
          description: draft.description || undefined,
          leader: draft.leader || undefined,
          members: draft.members.length ? draft.members : undefined,
        });
      }
      setModalOpen(false);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function removeTeam(t: Team) {
    if (!confirm(`确认删除团队 ${t.key}? 关联的领域负责人不会自动清除。`)) return;
    try {
      await deleteTeam(t.key);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function doAddMember(teamKey: string) {
    const val = (addMemberFor[teamKey] || "").trim();
    if (!val) return;
    try {
      await addTeamMember(teamKey, val);
      setAddMemberFor((m) => ({ ...m, [teamKey]: "" }));
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function doRemoveMember(teamKey: string, member: string) {
    if (!confirm(`从团队移除 ${member} ?`)) return;
    try {
      await removeTeamMember(teamKey, member);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title="团队管理"
      kicker="Teams & Domain Ownership"
      description="文档维护团队（负责人 + 成员）。负责人可直接拉人进团队。团队可被指定为「领域/分类」的负责人，负责维护该领域下文档结构与模块归属。"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div className="muted" style={{ fontSize: 13 }}>共 {teams?.length ?? 0} 个团队</div>
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
                <th>负责领域</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {(teams || []).map((t) => {
                const owned = ownedDomainsFor(t.key);
                const addVal = addMemberFor[t.key] || "";
                return (
                  <tr key={t.key}>
                    <td>
                      <div style={{ fontWeight: 640 }}>{t.name || t.key}</div>
                      <div className="muted" style={{ fontSize: 12 }}>{t.key}{t.description ? " · " + t.description : ""}</div>
                    </td>
                    <td>{t.leader ? <span className="badge badge-success">{t.leader}</span> : <span className="muted">—</span>}</td>
                    <td>
                      <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                        {(t.members || []).map((m) => (
                          <span key={m} className="tag">
                            {m}
                            <button style={{ border: 0, background: "transparent", cursor: "pointer", color: "inherit", opacity: 0.6, padding: 0, lineHeight: 0 }} onClick={() => doRemoveMember(t.key, m)} title="移除成员">
                              <X size={12} />
                            </button>
                          </span>
                        ))}
                        {(!t.members || t.members.length === 0) && <span className="muted" style={{ fontSize: 12 }}>—</span>}
                      </div>
                      <div style={{ marginTop: 6, display: "flex", gap: 6, alignItems: "center" }}>
                        <input
                          style={{ width: 150, height: 32, fontSize: 13 }}
                          placeholder="添加成员 username"
                          value={addVal}
                          onChange={(e) => setAddMemberFor((prev) => ({ ...prev, [t.key]: e.target.value }))}
                          onKeyDown={(e) => e.key === "Enter" && doAddMember(t.key)}
                        />
                        <button className="icon-btn" onClick={() => doAddMember(t.key)} title="拉人入队"><UserPlus size={14} /></button>
                      </div>
                    </td>
                    <td>
                      {owned.length > 0 ? owned.map((n) => <span key={n} className="tag" style={{ marginRight: 4 }}>{n}</span>) : <span className="muted" style={{ fontSize: 12 }}>未绑定领域</span>}
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
              {(teams || []).length === 0 ? (
                <tr><td colSpan={5}><div className="empty-state" style={{ minHeight: 170, border: 0, background: "transparent" }}>
                  <div>
                    <div style={{ fontWeight: 640, color: "hsl(var(--foreground))" }}>暂无团队</div>
                    <p style={{ marginTop: 4, fontSize: 13 }}>创建一个团队，并将其设置为某个领域的负责人，即可开始维护文档结构。</p>
                  </div>
                </div></td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={isEdit ? `编辑团队 · ${draft.key}` : "新增团队"}
        subtitle={isEdit ? "更新团队信息与成员" : "负责人会自动加入成员列表；Key 可用于领域负责人绑定"}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>取消</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim()}>
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
          <label>负责人 (username)</label>
          <input value={draft.leader} placeholder="如 alice" onChange={(e) => setDraft({ ...draft, leader: e.target.value })} />
        </div>
        <div className="field">
          <label>成员</label>
          <Combobox options={draft.members.map((m) => ({ value: m, label: m }))} value={draft.members} onChange={(members) => setDraft({ ...draft, members })} allowCreate placeholder="输入 username 回车添加…" />
          <span className="field-hint">负责人保存后会自动进入成员列表。</span>
        </div>
      </Modal>
    </AdminShell>
  );
}
