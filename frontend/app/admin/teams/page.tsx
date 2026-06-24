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
import { useI18n } from "@/lib/i18n";

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
  const { t } = useI18n();
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

  async function removeTeam(team: Team) {
    if (!confirm(t("admin.teams.confirm_deletion_of_team_value1_associated_category_owners", { value1: team.key }))) return;
    try {
      await deleteTeam(team.key);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title={t("component.adminShell.team_management")}
      kicker="Teams"
      description={t("admin.teams.documentation_maintenance_team_owner_members_the_owner_can")}
    >
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("admin.teams.search_team_name_id_owner_member")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> {t("admin.teams.add_team")}
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.teams.team")}</th>
                <th>{t("admin.teams.owner")}</th>
                <th>{t("admin.teams.members")}</th>
                <th>{t("admin.teams.category_owner")}</th>
                <th className="table-actions-col">{t("admin.modules.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {pageTeams.map((team) => {
                const owned = ownedDomainsFor(team.key);
                return (
                  <tr key={team.key}>
                    <td>
                      <div style={{ fontWeight: 640 }}>{team.name || t("admin.teams.unnamed_team")}</div>
                      {team.description ? <div className="muted" style={{ fontSize: 12 }}>{team.description}</div> : null}
                    </td>
                    <td>
                      {(team.leaders || []).length > 0 ? (
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                          {(team.leaders || []).map((l) => <span key={l} className="badge badge-success">{l}</span>)}
                        </div>
                      ) : (
                        <span className="muted">—</span>
                      )}
                    </td>
                    <td>
                      {(team.members || []).length > 0 ? (
                        <span className="tag">{(team.members || []).length} 人</span>
                      ) : (
                        <span className="muted" style={{ fontSize: 12 }}>—</span>
                      )}
                    </td>
                    <td>
                      {owned.length > 0 ? owned.map((n) => <span key={n} className="tag" style={{ marginRight: 4 }}>{n}</span>) : <span className="muted" style={{ fontSize: 12 }}>{t("admin.teams.unbound_category")}</span>}
                    </td>
                    <td className="table-actions-cell">
                      <div className="row-actions">
                        <button className="icon-btn" onClick={() => openEdit(team)} aria-label={t("admin.categories.edit")}><Pencil size={14} /></button>
                        <button className="icon-btn danger" onClick={() => removeTeam(team)} aria-label={t("admin.categories.delete")}><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {total === 0 ? (
                <tr><td colSpan={5}>
                  <EmptyState
                    icon={UsersRound}
                    title={keyword ? t("admin.teams.no_matching_teams") : t("admin.teams.no_teams")}
                    hint={keyword ? t("admin.mcpLogs.try_a_different_keyword") : t("admin.teams.create_a_team_and_assign_it_as_the")}
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
        title={isEdit ? t("admin.teams.edit_team_value1", { value1: draft.key }) : t("admin.teams.add_team")}
        subtitle={isEdit ? t("admin.teams.update_team_info_and_members") : t("admin.teams.owner_is_automatically_added_to_the_member_list")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{t("admin.categories.cancel")}</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim() || draft.leaders.length === 0}>
              {isEdit ? t("admin.modules.save") : t("admin.teams.create_team")}
            </button>
          </>
        }
      >
        <div className="field">
          <label>{t("admin.categories.name")}</label>
          <input value={draft.name} placeholder={t("admin.teams.e_g_r_and_d_specification_group")} autoFocus onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          {isEdit ? <span className="field-hint">{t("admin.categories.id")} <code className="code-chip">{draft.key}</code>{t("admin.categories.system_generated_immutable")}</span> : <span className="field-hint">{t("admin.categories.id_is_auto_generated_by_the_system")}</span>}
        </div>
        <div className="field">
          <label>{t("admin.categories.description")}</label>
          <input value={draft.description} placeholder={t("admin.teams.one_sentence_description_of_the_team_s_responsibilities")} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>
        <div className="field">
          <label>{t("admin.teams.owner_2")}</label>
          <div className="picker-field">
            {draft.leaders.length > 0 ? (
              draft.leaders.map((l) => <span key={l} className="tag">{l}</span>)
            ) : (
              <span className="muted" style={{ fontSize: 13 }}>{t("admin.teams.not_specified")}</span>
            )}
            <button type="button" className="button" onClick={() => setPicker("leader")}>
              <UserPlus size={14} /> {t("admin.teams.select_owner")}
            </button>
          </div>
          <span className="field-hint">{t("admin.teams.at_least_one_owner_multiple_allowed_manages_content")}</span>
        </div>
        <div className="field">
          <label>{t("admin.teams.members")}</label>
          <div className="picker-field">
            {draft.members.length > 0 ? (
              draft.members.map((m) => <span key={m} className="tag">{m}</span>)
            ) : (
              <span className="muted" style={{ fontSize: 13 }}>{t("admin.teams.no_members")}</span>
            )}
            <button type="button" className="button" onClick={() => setPicker("members")}>
              <UserPlus size={14} /> {t("admin.teams.add_members")}
            </button>
          </div>
        </div>
      </Modal>

      <Modal
        open={picker !== null}
        onClose={() => setPicker(null)}
        title={picker === "leader" ? t("admin.teams.select_owner") : t("admin.teams.add_members")}
        subtitle={picker === "leader" ? t("admin.teams.search_and_expand_by_department_select_at_least") : t("admin.teams.search_and_expand_multi_select_by_department")}
        footer={<button className="button button-primary" onClick={() => setPicker(null)}>{t("admin.modules.done")}</button>}
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
