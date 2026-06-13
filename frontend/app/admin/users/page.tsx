"use client";

import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Search, Trash2, Users as UsersIcon } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { Pagination } from "@/components/ui/pagination";
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/components/ui/empty-state";
import { createUser, deleteUser, getCategories, getTeams, getUsers, updateUser } from "@/lib/api";
import type { Category, Team, User } from "@/types/modex";

const PAGE_SIZE = 8;

type Draft = {
  id?: string;
  username: string;
  display_name: string;
  email: string;
  department: string;
  managed_categories: string[];
  is_super_admin: boolean;
  status: string;
};

const emptyDraft: Draft = {
  username: "",
  display_name: "",
  email: "",
  department: "",
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
    getTeams().then(setTeams).catch(() => {});
    getCategories().then(setCategories).catch(() => {});
  }, []);

  // Identity is a single derived tier: super admin > team leader > member.
  const leaderUsernames = useMemo(() => new Set((teams || []).map((t) => t.leader).filter(Boolean)), [teams]);
  function identityOf(u: User): { label: string; cls: string } {
    if (u.is_super_admin) return { label: "超级管理员", cls: "badge-danger" };
    if (leaderUsernames.has(u.username)) return { label: "团队负责人", cls: "badge-success" };
    return { label: "成员", cls: "" };
  }

  // Team membership is owned by the Team (leader + members), so a user's teams
  // are derived here rather than stored on the user.
  function teamsOf(u: User): string[] {
    return (teams || [])
      .filter((t) => t.leader === u.username || (t.members || []).includes(u.username))
      .map((t) => t.name || t.key);
  }

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
          managed_categories: draft.managed_categories,
          is_super_admin: draft.is_super_admin,
        });
      }
      setModalOpen(false);
      await refresh();
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
    <AdminShell title="用户管理" kicker="Users" description="管理用户资料与权限。团队归属在「团队管理」中维护；使用 OIDC 登录时用户会自动同步到此目录。">
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
                <th>团队</th>
                <th>状态</th>
                <th style={{ textAlign: "right" }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {pageUsers.map((u) => {
                const ident = identityOf(u);
                const userTeams = teamsOf(u);
                return (
                <tr key={u.id}>
                  <td><div style={{ fontWeight: 640 }}>{u.display_name || u.username}</div></td>
                  <td className="muted" style={{ fontSize: 13 }}>{u.email || "—"}</td>
                  <td>{u.department || "—"}</td>
                  <td><span className={`badge ${ident.cls}`}>{ident.label}</span></td>
                  <td>{userTeams.length ? userTeams.map((t) => <span className="tag" key={t} style={{ marginRight: 4 }}>{t}</span>) : <span className="muted" style={{ fontSize: 12 }}>—</span>}</td>
                  <td>
                    <span className={`badge ${u.status === "disabled" ? "badge-danger" : "badge-success"}`}>
                      <span className="badge-dot" />{u.status === "disabled" ? "已停用" : "启用"}
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
                <tr><td colSpan={7}>
                  <EmptyState
                    icon={UsersIcon}
                    title="暂无用户"
                    hint="点击右上角「新增用户」创建第一个用户；使用 OIDC 登录时用户会自动出现在这里。"
                  />
                </td></tr>
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
        subtitle={isEdit ? "更新用户资料与权限" : "创建一个新用户"}
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
            <input value={draft.department} placeholder="如 平台组" onChange={(e) => setDraft({ ...draft, department: e.target.value })} />
          </div>
        </div>
        <div className="field">
          <label>可管理分类</label>
          <Combobox options={categoryOptions} value={draft.managed_categories} onChange={(managed_categories) => setDraft({ ...draft, managed_categories })} placeholder="搜索分类…" />
          <span className="field-hint">该用户可管理所选分类下的内容。超级管理员可管理全部分类，无需在此指定。</span>
        </div>
        {isEdit ? (
          <div className="field">
            <Switch
              checked={draft.status !== "disabled"}
              onChange={(on) => setDraft({ ...draft, status: on ? "active" : "disabled" })}
              label="启用账号"
              hint="停用后该用户将无法登录。"
            />
          </div>
        ) : null}
        <div className="field">
          <Switch
            checked={draft.is_super_admin}
            onChange={(on) => setDraft({ ...draft, is_super_admin: on })}
            tone="danger"
            label="超级管理员"
            hint="拥有全部权限，可管理所有用户与分类。"
          />
        </div>
      </Modal>
    </AdminShell>
  );
}
