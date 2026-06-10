"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/admin-shell";
import { createUser, deleteUser, getCategories, getGroups, getUsers, updateUser } from "@/lib/api";
import type { Group, User } from "@/types/modex";

const ROLES = ["admin", "maintainer", "viewer"];

function parseList(value: string): string[] {
  return value
    .split(/[,\s]+/)
    .map((v) => v.trim())
    .filter(Boolean);
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [keyword, setKeyword] = useState("");
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState({ username: "", display_name: "", email: "", department: "", groups: "", roles: "viewer", managed: "", super_admin: false });
  const [platforms, setPlatforms] = useState<string[]>([]);

  async function refresh(kw = keyword) {
    try {
      setUsers(await getUsers(kw));
    } catch (e) {
      setError(String(e));
    }
  }

  useEffect(() => {
    refresh("");
    getGroups().then(setGroups).catch(() => {});
    getCategories()
      .then((tree) => {
        const ids: string[] = [];
        const walk = (nodes: typeof tree) => nodes.forEach((n) => { ids.push(n.id); if (n.children) walk(n.children); });
        walk(tree);
        setPlatforms(ids);
      })
      .catch(() => {});
  }, []);

  async function submitCreate() {
    setError("");
    try {
      await createUser({
        username: form.username,
        display_name: form.display_name,
        email: form.email,
        department: form.department,
        groups: parseList(form.groups),
        roles: parseList(form.roles),
        managed_categories: parseList(form.managed),
        is_super_admin: form.super_admin,
      });
      setForm({ username: "", display_name: "", email: "", department: "", groups: "", roles: "viewer", managed: "", super_admin: false });
      await refresh();
      getGroups().then(setGroups).catch(() => {});
    } catch (e) {
      setError(String(e));
    }
  }

  async function saveEdit() {
    if (!editing) return;
    setError("");
    try {
      await updateUser(editing.id, {
        display_name: editing.display_name,
        email: editing.email,
        department: editing.department,
        groups: editing.groups,
        roles: editing.roles,
        managed_categories: editing.managed_categories,
        status: editing.status
      });
      setEditing(null);
      await refresh();
      getGroups().then(setGroups).catch(() => {});
    } catch (e) {
      setError(String(e));
    }
  }

  async function remove(user: User) {
    if (!confirm(`确认删除用户 ${user.username}?`)) return;
    setError("");
    try {
      await deleteUser(user.id);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell title="用户管理" kicker="Users & Groups" description="管理用户身份、用户组和角色。OIDC 登录时用户与用户组会自动同步到此目录。">
      {error ? <div className="panel" style={{ borderColor: "#ef4444", color: "#b91c1c" }}>{error}</div> : null}

      <section className="panel">
        <h2 className="font-semibold">新增用户</h2>
        <div className="mt-3 grid gap-2" style={{ gridTemplateColumns: "repeat(3, minmax(0, 1fr))" }}>
          <input className="input" placeholder="用户名*" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
          <input className="input" placeholder="显示名" value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} />
          <input className="input" placeholder="邮箱" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          <input className="input" placeholder="部门" value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} />
          <input className="input" placeholder="用户组(逗号分隔)" value={form.groups} onChange={(e) => setForm({ ...form, groups: e.target.value })} />
          <input className="input" placeholder="角色(逗号分隔)" value={form.roles} onChange={(e) => setForm({ ...form, roles: e.target.value })} />
          <input className="input" placeholder="可管理平台 ID(逗号分隔)" value={form.managed} onChange={(e) => setForm({ ...form, managed: e.target.value })} />
        </div>
        <label className="mt-2 flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.super_admin}
            onChange={(e) => setForm({ ...form, super_admin: e.target.checked })}
          />
          超级管理员（拥有全部权限，可管理所有用户与领域）
        </label>
        <div className="mt-2 flex gap-2">
          <button className="button button-primary" onClick={submitCreate} disabled={!form.username}>创建</button>
          <span className="muted text-xs" style={{ alignSelf: "center" }}>可选平台: {platforms.join(" / ") || "无"}</span>
        </div>
      </section>

      <section className="panel">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold">用户列表（{users.length}）</h2>
          <div className="flex gap-2">
            <input className="input" placeholder="搜索用户名/邮箱/部门" value={keyword} onChange={(e) => setKeyword(e.target.value)} />
            <button className="button" onClick={() => refresh()}>搜索</button>
          </div>
        </div>
        <table className="data-table mt-3">
          <thead>
            <tr>
              <th>用户</th>
              <th>部门</th>
              <th>用户组</th>
              <th>角色</th>
              <th>可管理平台</th>
              <th>来源</th>
              <th>状态</th>
              <th>最近登录</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>
                  <div className="font-semibold">{u.display_name}</div>
                  <div className="muted text-xs">{u.username} · {u.email || "—"}</div>
                </td>
                <td>{u.department || "—"}</td>
                <td>{(u.groups || []).map((g) => <span className="tag mr-1" key={g}>{g}</span>)}</td>
                <td>{(u.roles || []).map((r) => <span className="tag mr-1" key={r}>{r}</span>)}</td>
                <td>{u.is_super_admin ? <span className="tag">全部（超管）</span> : (u.managed_categories || []).map((c) => <span className="tag mr-1" key={c}>{c}</span>)}</td>
                <td>{u.source || "—"}</td>
                <td><span className="status-dot mr-2" />{u.status || "active"}</td>
                <td className="muted text-xs">{u.last_login_at && !u.last_login_at.startsWith("0001") ? u.last_login_at.slice(0, 19).replace("T", " ") : "—"}</td>
                <td>
                  <button className="button" onClick={() => setEditing({ ...u, groups: u.groups || [], roles: u.roles || [], managed_categories: u.managed_categories || [] })}>编辑</button>
                  <button className="button ml-1" onClick={() => remove(u)}>删除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {editing ? (
        <section className="panel">
          <h2 className="font-semibold">编辑用户 · {editing.username}</h2>
          <div className="mt-3 grid gap-2" style={{ gridTemplateColumns: "repeat(2, minmax(0, 1fr))" }}>
            <label className="grid gap-1 text-sm">显示名
              <input className="input" value={editing.display_name} onChange={(e) => setEditing({ ...editing, display_name: e.target.value })} />
            </label>
            <label className="grid gap-1 text-sm">邮箱
              <input className="input" value={editing.email} onChange={(e) => setEditing({ ...editing, email: e.target.value })} />
            </label>
            <label className="grid gap-1 text-sm">部门
              <input className="input" value={editing.department} onChange={(e) => setEditing({ ...editing, department: e.target.value })} />
            </label>
            <label className="grid gap-1 text-sm">用户组(逗号分隔)
              <input className="input" value={(editing.groups || []).join(", ")} onChange={(e) => setEditing({ ...editing, groups: parseList(e.target.value) })} />
            </label>
            <label className="grid gap-1 text-sm">可管理平台 ID(逗号分隔)
              <input className="input" value={(editing.managed_categories || []).join(", ")} onChange={(e) => setEditing({ ...editing, managed_categories: parseList(e.target.value) })} />
            </label>
            <label className="grid gap-1 text-sm">状态
              <select className="input" value={editing.status || "active"} onChange={(e) => setEditing({ ...editing, status: e.target.value })}>
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
            </label>
            <div className="grid gap-1 text-sm">角色
              <div className="flex gap-3">
                {ROLES.map((role) => (
                  <label key={role} className="flex items-center gap-1">
                    <input
                      type="checkbox"
                      style={{ width: "auto", height: "auto" }}
                      checked={(editing.roles || []).includes(role)}
                      onChange={(e) => {
                        const set = new Set(editing.roles || []);
                        if (e.target.checked) set.add(role); else set.delete(role);
                        setEditing({ ...editing, roles: [...set] });
                      }}
                    />
                    {role}
                  </label>
                ))}
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={!!editing.is_super_admin}
                onChange={(e) => setEditing({ ...editing, is_super_admin: e.target.checked })}
              />
              超级管理员（拥有全部权限，可管理所有用户与领域）
            </label>
          </div>
          <div className="mt-3 flex gap-2">
            <button className="button button-primary" onClick={saveEdit}>保存</button>
            <button className="button" onClick={() => setEditing(null)}>取消</button>
          </div>
        </section>
      ) : null}
    </AdminShell>
  );
}
