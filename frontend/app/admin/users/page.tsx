"use client";

import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Search, Trash2 } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { Pagination } from "@/components/ui/pagination";
import { createUser, deleteUser, getCategories, getGroups, getTeams, getUsers, updateUser } from "@/lib/api";
import type { Category, Group, Team, User } from "@/types/modex";

const ROLES = ["admin", "maintainer", "viewer"];
const PAGE_SIZE = 8;

type Draft = {
  id?: string;
  username: string;
  display_name: string;
  email: string;
  department: string;
  groups: string[];
  roles: string[];
  managed_categories: string[];
  is_super_admin: boolean;
  status: string;
};

const emptyDraft: Draft = {
  username: "",
  display_name: "",
  email: "",
  department: "",
  groups: [],
  roles: ["viewer"],
  managed_categories: [],
  is_super_admin: false,
  status: "active",
};

function flattenCategories(tree: Category[]): ComboOption[] {
  const out: ComboOption[] = [];
  const walk = (nodes: Category[], depth: number) =>
    nodes.forEach((n) => {
      out.push({ value: n.id, label: n.name, hint: n.key, depth });
      if (n.children) walk(n.children, depth + 1);
    });
  walk(tree, 0);
  return out;
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const isEdit = !!draft.id;

  async function refresh(kw = keyword) {
    try {
      setUsers(await getUsers(kw));
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }

  useEffect(() => {
    refresh("");
    getGroups().then(setGroups).catch(() => {});
    getTeams().then(setTeams).catch(() => {});
    getCategories().then(setCategories).catch(() => {});
  }, []);

  // Identity is a single derived tier: super admin > team leader > regular user.
  const leaderUsernames = useMemo(() => new Set((teams || []).map((t) => t.leader).filter(Boolean)), [teams]);
  function identityOf(u: User): { label: string; cls: string } {
    if (u.is_super_admin) return { label: "超级管理员", cls: "badge-danger" };
    if (leaderUsernames.has(u.username)) return { label: "团队负责人", cls: "badge-success" };
    return { label: "普通用户", cls: "" };
  }

  const groupOptions: ComboOption[] = useMemo(
    () => groups.map((g) => ({ value: g.group_key, label: g.name || g.group_key, hint: g.group_key })),
    [groups],
  );
  const roleOptions: ComboOption[] = ROLES.map((r) => ({ value: r, label: r }));
  const categoryOptions = useMemo(() => flattenCategories(categories), [categories]);

  const pageUsers = users.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  function openCreate() {
    setDraft(emptyDraft);
    setModalOpen(true);
  }
  function openEdit(u: User) {
    setDraft({
      id: u.id,
      username: u.username,
      display_name: u.display_name || "",
      email: u.email || "",
      department: u.department || "",
      groups: u.groups || [],
      roles: u.roles || [],
      managed_categories: u.managed_categories || [],
      is_super_admin: !!u.is_super_admin,
      status: u.status || "active",
    });
    setModalOpen(true);
  }

  async function submit() {
    setError("");
    try {
      if (isEdit) {
        await updateUser(draft.id!, {
          display_name: draft.display_name,
          email: draft.email,
          department: draft.department,
          groups: draft.groups,
          roles: draft.roles,
          managed_categories: draft.managed_categories,
          is_super_admin: draft.is_super_admin,
          status: draft.status,
        });
      } else {
        await createUser({
          username: draft.username,
          display_name: draft.display_name,
          email: draft.email,
          department: draft.department,
          groups: draft.groups,
          roles: draft.roles,
          managed_categories: draft.managed_categories,
          is_super_admin: draft.is_super_admin,
        });
      }
      setModalOpen(false);
      await refresh();
      getGroups().then(setGroups).catch(() => {});
    } catch (e) {
      setError(String(e));
    }
  }

  async function remove(user: User) {
    if (!confirm(`确认删除用户 ${user.username}?`)) return;
    try {
      await deleteUser(user.id);
      await refresh();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell title="用户管理" kicker="Users & Groups" description="管理用户身份、用户组和角色。OIDC 登录时用户与用户组会自动同步到此目录。">
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder="搜索显示名 / 邮箱 / 部门"
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
            onKeyDown={(e) => e.key === "Enter" && refresh()}
          />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> 新增用户
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>用户</th>
                <th>邮箱</th>
                <th>部门</th>
                <th>身份</th>
                <th>用户组</th>
                <th>状态</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {pageUsers.map((u) => {
                const ident = identityOf(u);
                return (
                <tr key={u.id}>
                  <td><div style={{ fontWeight: 640 }}>{u.display_name || u.username}</div></td>
                  <td className="muted" style={{ fontSize: 13 }}>{u.email || "—"}</td>
                  <td>{u.department || "—"}</td>
                  <td><span className={`badge ${ident.cls}`}>{ident.label}</span></td>
                  <td>{(u.groups || []).length ? (u.groups || []).map((g) => <span className="tag" key={g} style={{ marginRight: 4 }}>{g}</span>) : <span className="muted" style={{ fontSize: 12 }}>—</span>}</td>
                  <td>
                    <span className={`badge ${u.status === "disabled" ? "badge-danger" : "badge-success"}`}>
                      <span className="badge-dot" />{u.status || "active"}
                    </span>
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="icon-btn" onClick={() => openEdit(u)} aria-label="编辑"><Pencil size={14} /></button>
                      <button className="icon-btn danger" onClick={() => remove(u)} aria-label="删除"><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
                );
              })}
              {!pageUsers.length ? (
                <tr><td colSpan={7}><div className="empty-state" style={{ minHeight: 160, border: 0, background: "transparent" }}>暂无用户</div></td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
        <Pagination page={page} pageSize={PAGE_SIZE} total={users.length} onPage={setPage} />
      </div>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={isEdit ? `编辑用户 · ${draft.username}` : "新增用户"}
        subtitle={isEdit ? "更新用户资料、用户组与权限" : "创建一个新的平台用户"}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>取消</button>
            <button className="button button-primary" onClick={submit} disabled={!isEdit && !draft.username.trim()}>
              {isEdit ? "保存" : "创建"}
            </button>
          </>
        }
      >
        <div className="field-row">
          <div className="field">
            <label>用户名{isEdit ? "" : " *"}</label>
            <input value={draft.username} disabled={isEdit} placeholder="如 alice" onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
          </div>
          <div className="field">
            <label>显示名</label>
            <input value={draft.display_name} placeholder="如 Alice" onChange={(e) => setDraft({ ...draft, display_name: e.target.value })} />
          </div>
        </div>
        <div className="field-row">
          <div className="field">
            <label>邮箱</label>
            <input value={draft.email} placeholder="name@example.com" onChange={(e) => setDraft({ ...draft, email: e.target.value })} />
          </div>
          <div className="field">
            <label>部门</label>
            <input value={draft.department} placeholder="如 工程化" onChange={(e) => setDraft({ ...draft, department: e.target.value })} />
          </div>
        </div>
        <div className="field">
          <label>用户组</label>
          <Combobox options={groupOptions} value={draft.groups} onChange={(groups) => setDraft({ ...draft, groups })} allowCreate placeholder="搜索或新增用户组…" />
        </div>
        <div className="field">
          <label>角色</label>
          <Combobox options={roleOptions} value={draft.roles} onChange={(roles) => setDraft({ ...draft, roles })} placeholder="选择角色…" />
        </div>
        <div className="field">
          <label>可管理平台 / 领域</label>
          <Combobox options={categoryOptions} value={draft.managed_categories} onChange={(managed_categories) => setDraft({ ...draft, managed_categories })} placeholder="搜索能力域…" />
          <span className="field-hint">超级管理员可管理全部领域，无需在此指定。</span>
        </div>
        {isEdit ? (
          <div className="field">
            <label>状态</label>
            <Combobox options={[{ value: "active", label: "active" }, { value: "disabled", label: "disabled" }]} value={[draft.status]} onChange={(v) => setDraft({ ...draft, status: v[0] || "active" })} multiple={false} />
          </div>
        ) : null}
        <label className="field" style={{ flexDirection: "row", alignItems: "center", gap: 10 }}>
          <input type="checkbox" style={{ width: "auto", height: "auto" }} checked={draft.is_super_admin} onChange={(e) => setDraft({ ...draft, is_super_admin: e.target.checked })} />
          <span style={{ fontSize: 13 }}>超级管理员（拥有全部权限，可管理所有用户与领域）</span>
        </label>
      </Modal>
    </AdminShell>
  );
}
