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
    if (u.is_super_admin) return { label: t("legacy.56db248412ff"), cls: "badge-danger" };
    if (leaderUsernames.has(u.username)) return { label: t("legacy.d59c8cdb38a5"), cls: "badge-success" };
    return { label: t("legacy.6e6d6ddbb7c1"), cls: "" };
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
    if (!confirm(t("legacy.5d1e06d73d56", { value1: user.username }))) return;
    try {
      await deleteUser(user.id);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell title={t("legacy.fbf413d429bd")} kicker="Users" description={t("legacy.f40dbd142ee2")}>
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("legacy.dbff39a5e9ba")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> {t("legacy.ebabc83b6830")}
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.0d0e1a86b3aa")}</th>
                <th>{t("legacy.73075237fd0f")}</th>
                <th>{t("legacy.f128cdf1dae2")}</th>
                <th>{t("legacy.2a9c9e997642")}</th>
                <th>{t("legacy.acdf17f4e9c4")}</th>
                <th>{t("legacy.6320b4a8722a")}</th>
                <th style={{ textAlign: "right" }}>{t("legacy.ed31fbb483ee")}</th>
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
                      <span className="badge-dot" />{u.status === "disabled" ? t("legacy.a8c3698b5b8c") : t("legacy.f4f0ead1116b")}
                    </span>
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="icon-btn" onClick={() => openEdit(u)} aria-label={t("legacy.051836569928")}><Pencil size={14} /></button>
                      <button className="icon-btn danger" onClick={() => remove(u)} aria-label={t("legacy.2f9daa828907")}><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
                );
              })}
              {!pageUsers.length ? (
                <tr><td colSpan={7}>
                  <EmptyState
                    icon={UsersIcon}
                    title={t("legacy.a56b93c27aec")}
                    hint={t("legacy.5748c7aebb94")}
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
        title={isEdit ? t("legacy.67660891683d", { value1: draft.username }) : t("legacy.ebabc83b6830")}
        subtitle={isEdit ? t("legacy.82243461e51b") : t("legacy.f57b5927ab88")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{t("legacy.2cd0f3be8738")}</button>
            <button className="button button-primary" onClick={submit} disabled={!isEdit && !draft.username.trim()}>
              {isEdit ? t("legacy.a3030bf8f16d") : t("legacy.cde2cd071d25")}
            </button>
          </>
        }
      >
        <div className="field-row">
          <div className="field">
            <label>{t("legacy.1a3f0617d6de")}{isEdit ? "" : " *"}</label>
            <input value={draft.username} disabled={isEdit} placeholder={t("legacy.2599cb70de43")} onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("legacy.4587cc06a981")}</label>
            <input value={draft.display_name} placeholder={t("legacy.7ffc934b2329")} onChange={(e) => setDraft({ ...draft, display_name: e.target.value })} />
          </div>
        </div>
        <div className="field-row">
          <div className="field">
            <label>{t("legacy.73075237fd0f")}</label>
            <input value={draft.email} placeholder="name@example.com" onChange={(e) => setDraft({ ...draft, email: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("legacy.f128cdf1dae2")}</label>
            <input value={draft.department} placeholder={t("legacy.26fc3414b9fd")} onChange={(e) => setDraft({ ...draft, department: e.target.value })} />
          </div>
        </div>
        <div className="field">
          <label>{t("legacy.2d5db2e77279")}</label>
          <Combobox options={categoryOptions} value={draft.managed_categories} onChange={(managed_categories) => setDraft({ ...draft, managed_categories })} placeholder={t("legacy.85e5f7b44180")} />
          <span className="field-hint">{t("legacy.b3634089a140")}</span>
        </div>
        {isEdit ? (
          <div className="field">
            <Switch
              checked={draft.status !== "disabled"}
              onChange={(on) => setDraft({ ...draft, status: on ? "active" : "disabled" })}
              label={t("legacy.10027ec29566")}
              hint={t("legacy.40160bbaa368")}
            />
          </div>
        ) : null}
        <div className="field">
          <Switch
            checked={draft.is_super_admin}
            onChange={(on) => setDraft({ ...draft, is_super_admin: on })}
            tone="danger"
            label={t("legacy.56db248412ff")}
            hint={t("legacy.f2d695d24fbe")}
          />
        </div>
      </Modal>
    </AdminShell>
  );
}
