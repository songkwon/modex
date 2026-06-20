"use client";

import { useEffect, useMemo, useState } from "react";
import { Pencil, Plus, Search, Trash2, Users as UsersIcon } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import { Pagination } from "@/components/ui/pagination";
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/components/ui/empty-state";
import { createUser, deleteUser, getCategories, getTeams, updateUser } from "@/lib/api";
import { usePaged } from "@/lib/use-paged";
import type { Category, Team, User } from "@/types/modex";
import { useI18n } from "@/lib/i18n";

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
  const { t } = useI18n();
  const [teams, setTeams] = useState<Team[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [keyword, setKeyword] = useState("");
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const isEdit = !!draft.id;

  const { items: pageUsers, total, page, setPage, error: loadError, reload } = usePaged<User>(
    "/api/admin/users",
    PAGE_SIZE,
    keyword.trim(),
  );

  // Teams power the derived identity/membership columns and must be the full
  // list (not paginated), so they are fetched separately without ?page=.
  useEffect(() => {
    getTeams().then(setTeams).catch(() => {});
    getCategories().then(setCategories).catch(() => {});
  }, []);

  // Identity is a single derived tier: super admin > team leader > member.
  const leaderUsernames = useMemo(() => new Set((teams || []).flatMap((t) => t.leaders || [])), [teams]);
  function identityOf(u: User): { label: string; cls: string } {
    if (u.is_super_admin) return { label: t("admin.users.superAdmin"), cls: "badge-danger" };
    if (leaderUsernames.has(u.username)) return { label: t("admin.users.team_lead"), cls: "badge-success" };
    return { label: t("admin.teams.members"), cls: "" };
  }

  // Team membership is owned by the Team (leader + members), so a user's teams
  // are derived here rather than stored on the user.
  function teamsOf(u: User): string[] {
    return (teams || [])
      .filter((t) => (t.leaders || []).includes(u.username) || (t.members || []).includes(u.username))
      .map((t) => t.name || t.key);
  }

  const categoryOptions = useMemo(() => flattenCategories(categories), [categories]);

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
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  async function remove(user: User) {
    if (!confirm(t("admin.users.confirm_deletion_of_user_value1", { value1: user.username }))) return;
    try {
      await deleteUser(user.id);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell title={t("component.adminShell.user_management")} kicker="Users" description={t("admin.users.manage_user_profiles_and_permissions_team_membership_is")}>
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("admin.users.search_display_name_email_department")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> {t("admin.users.add_user")}
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.mcpLogs.user")}</th>
                <th>{t("admin.users.email")}</th>
                <th>{t("admin.users.department")}</th>
                <th>{t("admin.users.identity")}</th>
                <th>{t("admin.teams.team")}</th>
                <th>{t("admin.releases.status")}</th>
                <th className="table-actions-col">{t("admin.modules.actions")}</th>
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
                      <span className="badge-dot" />{u.status === "disabled" ? t("admin.users.disabled") : t("admin.connectedApps.enable")}
                    </span>
                  </td>
                  <td className="table-actions-cell">
                    <div className="row-actions">
                      <button className="icon-btn" onClick={() => openEdit(u)} aria-label={t("admin.categories.edit")}><Pencil size={14} /></button>
                      <button className="icon-btn danger" onClick={() => remove(u)} aria-label={t("admin.categories.delete")}><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
                );
              })}
              {!pageUsers.length ? (
                <tr><td colSpan={7}>
                  <EmptyState
                    icon={UsersIcon}
                    title={t("admin.users.no_users")}
                    hint={t("admin.users.click_add_user_top_right_to_create_your")}
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
        title={isEdit ? t("admin.users.edit_user_value1", { value1: draft.username }) : t("admin.users.add_user")}
        subtitle={isEdit ? t("admin.users.update_user_profile_and_permissions") : t("admin.users.create_new_user")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{t("admin.categories.cancel")}</button>
            <button className="button button-primary" onClick={submit} disabled={!isEdit && !draft.username.trim()}>
              {isEdit ? t("admin.modules.save") : t("admin.categories.create")}
            </button>
          </>
        }
      >
        <div className="field-row">
          <div className="field">
            <label>{t("admin.users.username")}{isEdit ? "" : " *"}</label>
            <input value={draft.username} disabled={isEdit} placeholder={t("admin.users.e_g_alice")} onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("admin.users.display_name")}</label>
            <input value={draft.display_name} placeholder={t("admin.users.e_g_alice_2")} onChange={(e) => setDraft({ ...draft, display_name: e.target.value })} />
          </div>
        </div>
        <div className="field-row">
          <div className="field">
            <label>{t("admin.users.email")}</label>
            <input value={draft.email} placeholder="name@example.com" onChange={(e) => setDraft({ ...draft, email: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("admin.users.department")}</label>
            <input value={draft.department} placeholder={t("admin.users.e_g_platform_group")} onChange={(e) => setDraft({ ...draft, department: e.target.value })} />
          </div>
        </div>
        <div className="field">
          <label>{t("admin.users.manageable_categories")}</label>
          <Combobox options={categoryOptions} value={draft.managed_categories} onChange={(managed_categories) => setDraft({ ...draft, managed_categories })} placeholder={t("admin.users.search_category")} />
          <span className="field-hint">{t("admin.users.this_user_can_manage_content_under_the_selected")}</span>
        </div>
        {isEdit ? (
          <div className="field">
            <Switch
              checked={draft.status !== "disabled"}
              onChange={(on) => setDraft({ ...draft, status: on ? "active" : "disabled" })}
              label={t("admin.users.enable_account")}
              hint={t("admin.users.after_deactivation_this_user_cannot_log_in")}
            />
          </div>
        ) : null}
        <div className="field">
          <Switch
            checked={draft.is_super_admin}
            onChange={(on) => setDraft({ ...draft, is_super_admin: on })}
            tone="danger"
            label={t("admin.users.superAdmin")}
            hint={t("admin.users.has_full_permissions_and_can_manage_all_users")}
          />
        </div>
      </Modal>
    </AdminShell>
  );
}
