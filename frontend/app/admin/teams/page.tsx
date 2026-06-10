"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
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

function parseList(value: string): string[] {
  return value
    .split(/[,\s]+/)
    .map((v) => v.trim())
    .filter(Boolean);
}

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

export default function AdminTeamsPage() {
  const [teams, setTeams] = useState<Team[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<Team | null>(null);

  // Create form
  const [form, setForm] = useState({
    key: "",
    name: "",
    description: "",
    leader: "",
    members: "",
  });

  // Per-row quick add member
  const [addMemberFor, setAddMemberFor] = useState<Record<string, string>>({});

  async function refresh() {
    try {
      const [ts, tree] = await Promise.all([getTeams(), getCategories()]);
      setTeams(ts || []);
      setCategories(tree || []);
    } catch (e) {
      setError(String(e));
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  const flatCats = flattenCategories(categories);

  function ownedDomainsFor(teamKey: string): string[] {
    return flatCats
      .filter((c) => c.responsible_team === teamKey)
      .map((c) => c.name);
  }

  async function submitCreate() {
    setError("");
    try {
      const members = parseList(form.members);
      await createTeam({
        key: form.key,
        name: form.name || undefined,
        description: form.description || undefined,
        leader: form.leader || undefined,
        members: members.length ? members : undefined,
      });
      setForm({ key: "", name: "", description: "", leader: "", members: "" });
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function saveEdit() {
    if (!editing) return;
    setError("");
    try {
      await updateTeam(editing.key, {
        name: editing.name,
        description: editing.description,
        leader: editing.leader,
        members: editing.members,
      });
      setEditing(null);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  async function removeTeam(t: Team) {
    if (!confirm(`确认删除团队 ${t.key}? 关联的领域负责人不会自动清除。`)) return;
    setError("");
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
    setError("");
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
    setError("");
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
      description="文档维护团队（负责人 + 成员）。负责人可直接拉人进团队。团队可被指定为「领域/分类」的负责人，负责维护该领域下文档结构与模块归属。支持任意层级（参考 Mintlify / GitBook）。"
    >
      {error ? (
        <div className="panel" style={{ borderColor: "#ef4444", color: "#b91c1c" }}>
          {error}
        </div>
      ) : null}

      {/* Create */}
      <section className="panel">
        <h2 className="font-semibold">新增团队</h2>
        <div className="mt-3 grid gap-2" style={{ gridTemplateColumns: "repeat(5, minmax(0, 1fr))" }}>
          <input
            className="input"
            placeholder="Key * (唯一，如 standards)"
            value={form.key}
            onChange={(e) => setForm({ ...form, key: e.target.value })}
          />
          <input
            className="input"
            placeholder="名称"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <input
            className="input"
            placeholder="描述"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
          <input
            className="input"
            placeholder="负责人 (username)"
            value={form.leader}
            onChange={(e) => setForm({ ...form, leader: e.target.value })}
          />
          <input
            className="input"
            placeholder="初始成员 (逗号分隔)"
            value={form.members}
            onChange={(e) => setForm({ ...form, members: e.target.value })}
          />
        </div>
        <div className="mt-3 flex gap-2">
          <button className="button button-primary" onClick={submitCreate} disabled={!form.key}>
            创建团队
          </button>
          <span className="muted text-xs self-center">
            负责人会自动加入成员列表。Key 可用于 OwnerGroup / 领域负责人绑定。
          </span>
        </div>
      </section>

      {/* List */}
      <section className="panel">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold">团队列表（{teams?.length ?? 0}）</h2>
          <button className="button" onClick={refresh}>
            刷新
          </button>
        </div>

        {(teams || []).length === 0 && !error && (
          <div className="empty-state mt-3">
            <div>
              <div className="font-semibold text-foreground">暂无团队</div>
              <p className="mt-1 text-sm">创建一个团队，并将其设置为某个领域的负责人，即可开始维护文档结构。</p>
            </div>
          </div>
        )}

        {(teams || []).length > 0 && (
        <table className="data-table mt-3">
          <thead>
            <tr>
              <th>团队</th>
              <th>负责人</th>
              <th>成员</th>
              <th>负责领域</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {(teams || []).map((t) => {
              const owned = ownedDomainsFor(t.key);
              const addVal = addMemberFor[t.key] || "";
              return (
                <tr key={t.key}>
                  <td>
                    <div className="font-semibold">{t.name}</div>
                    <div className="muted text-xs">
                      {t.key} {t.description ? "· " + t.description : ""}
                    </div>
                  </td>
                  <td>
                    <span className="tag">{t.leader || "—"}</span>
                  </td>
                  <td>
                    <div className="flex flex-wrap gap-1">
                      {(t.members || []).map((m) => (
                        <span key={m} className="tag flex items-center gap-1">
                          {m}
                          <button
                            className="text-[10px] opacity-60 hover:opacity-100"
                            onClick={() => doRemoveMember(t.key, m)}
                            title="移除成员"
                          >
                            ×
                          </button>
                        </span>
                      ))}
                      {(!t.members || t.members.length === 0) && <span className="muted text-xs">—</span>}
                    </div>
                    {/* Quick add member - "负责人拉人" */}
                    <div className="mt-1 flex gap-1">
                      <input
                        className="input text-xs"
                        style={{ width: 120 }}
                        placeholder="添加成员 username"
                        value={addVal}
                        onChange={(e) =>
                          setAddMemberFor((prev) => ({ ...prev, [t.key]: e.target.value }))
                        }
                        onKeyDown={(e) => {
                          if (e.key === "Enter") doAddMember(t.key);
                        }}
                      />
                      <button className="button text-xs" onClick={() => doAddMember(t.key)}>
                        拉人
                      </button>
                    </div>
                  </td>
                  <td>
                    {owned.length > 0 ? (
                      owned.map((n) => (
                        <span key={n} className="tag mr-1 bg-emerald-100 text-emerald-700">
                          {n}
                        </span>
                      ))
                    ) : (
                      <span className="muted text-xs">未绑定领域</span>
                    )}
                  </td>
                  <td>
                    <button
                      className="button"
                      onClick={() =>
                        setEditing({
                          ...t,
                          members: t.members || [],
                        })
                      }
                    >
                      编辑
                    </button>
                    <button className="button ml-1" onClick={() => removeTeam(t)}>
                      删除
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
        )}

        <div className="muted text-xs mt-2">
          提示：只有超级管理员可以查看/管理团队。使用 <code>SUPER_ADMIN_USERS=yourname</code> 环境变量 + Mock 登录（或 OIDC）获得第一个超管身份后，即可在此创建团队并绑定领域。负责人可直接为团队“拉人”。
        </div>
      </section>

      {/* Edit panel */}
      {editing ? (
        <section className="panel">
          <h2 className="font-semibold">编辑团队 · {editing.key}</h2>
          <div className="mt-3 grid gap-3" style={{ gridTemplateColumns: "repeat(2, minmax(0, 1fr))" }}>
            <label className="grid gap-1 text-sm">
              名称
              <input
                className="input"
                value={editing.name}
                onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              />
            </label>
            <label className="grid gap-1 text-sm">
              描述
              <input
                className="input"
                value={editing.description || ""}
                onChange={(e) => setEditing({ ...editing, description: e.target.value })}
              />
            </label>
            <label className="grid gap-1 text-sm">
              负责人 (leader)
              <input
                className="input"
                value={editing.leader || ""}
                onChange={(e) => setEditing({ ...editing, leader: e.target.value })}
              />
            </label>
            <label className="grid gap-1 text-sm">
              成员 (逗号或空格分隔，保存时同步)
              <input
                className="input"
                value={(editing.members || []).join(", ")}
                onChange={(e) =>
                  setEditing({ ...editing, members: parseList(e.target.value) })
                }
              />
            </label>
          </div>

          <div className="mt-3 flex gap-2">
            <button className="button button-primary" onClick={saveEdit}>
              保存
            </button>
            <button className="button" onClick={() => setEditing(null)}>
              取消
            </button>
            <span className="muted text-xs self-center">
              保存后负责人会自动进入成员列表。
            </span>
          </div>
        </section>
      ) : null}
    </AdminShell>
  );
}
