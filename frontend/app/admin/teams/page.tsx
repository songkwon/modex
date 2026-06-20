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
    if (!confirm(t("legacy.5c2909021434", { value1: team.key }))) return;
    try {
      await deleteTeam(team.key);
      reload();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title={t("legacy.95a792201917")}
      kicker="Teams"
      description={t("legacy.d6c0c303d626")}
    >
      {(error || loadError) ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error || loadError}</div> : null}

      <div className="admin-toolbar">
        <div className="search-inline">
          <Search size={15} />
          <input
            placeholder={t("legacy.ba20d851f069")}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> {t("legacy.5fae8485fa3b")}
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.acdf17f4e9c4")}</th>
                <th>{t("legacy.a525346277ed")}</th>
                <th>{t("legacy.6e6d6ddbb7c1")}</th>
                <th>{t("legacy.1792a32bddf9")}</th>
                <th style={{ textAlign: "right" }}>{t("legacy.ed31fbb483ee")}</th>
              </tr>
            </thead>
            <tbody>
              {pageTeams.map((team) => {
                const owned = ownedDomainsFor(team.key);
                return (
                  <tr key={team.key}>
                    <td>
                      <div style={{ fontWeight: 640 }}>{team.name || team.key}</div>
                      <div className="muted" style={{ fontSize: 12 }}>{team.key}{team.description ? " · " + team.description : ""}</div>
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
                      {owned.length > 0 ? owned.map((n) => <span key={n} className="tag" style={{ marginRight: 4 }}>{n}</span>) : <span className="muted" style={{ fontSize: 12 }}>{t("legacy.5c574cbbe0e5")}</span>}
                    </td>
                    <td>
                      <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                        <button className="icon-btn" onClick={() => openEdit(team)} aria-label={t("legacy.051836569928")}><Pencil size={14} /></button>
                        <button className="icon-btn danger" onClick={() => removeTeam(team)} aria-label={t("legacy.2f9daa828907")}><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {total === 0 ? (
                <tr><td colSpan={5}>
                  <EmptyState
                    icon={UsersRound}
                    title={keyword ? t("legacy.16b8cd642237") : t("legacy.f36b68f045df")}
                    hint={keyword ? t("legacy.018f0b4a413c") : t("legacy.d517ea4d7b43")}
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
        title={isEdit ? t("legacy.ba200870b207", { value1: draft.key }) : t("legacy.5fae8485fa3b")}
        subtitle={isEdit ? t("legacy.8b4fd28534d8") : t("legacy.af6d8fbf9d33")}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{t("legacy.2cd0f3be8738")}</button>
            <button className="button button-primary" onClick={submit} disabled={!draft.name.trim() || draft.leaders.length === 0}>
              {isEdit ? t("legacy.a3030bf8f16d") : t("legacy.d2ec1c315266")}
            </button>
          </>
        }
      >
        <div className="field">
          <label>{t("legacy.909712f2847d")}</label>
          <input value={draft.name} placeholder={t("legacy.49b7511568e2")} autoFocus onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          {isEdit ? <span className="field-hint">{t("legacy.301c9461c1d0")} <code className="code-chip">{draft.key}</code>{t("legacy.e7d3ed9636ac")}</span> : <span className="field-hint">{t("legacy.de9b50b07a8c")}</span>}
        </div>
        <div className="field">
          <label>{t("legacy.dc2ba467fc7a")}</label>
          <input value={draft.description} placeholder={t("legacy.948180f4b4ac")} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>
        <div className="field">
          <label>{t("legacy.d196cf0f1ea8")}</label>
          <div className="picker-field">
            {draft.leaders.length > 0 ? (
              draft.leaders.map((l) => <span key={l} className="tag">{l}</span>)
            ) : (
              <span className="muted" style={{ fontSize: 13 }}>{t("legacy.7409a608060f")}</span>
            )}
            <button type="button" className="button" onClick={() => setPicker("leader")}>
              <UserPlus size={14} /> {t("legacy.5b894d63abb8")}
            </button>
          </div>
          <span className="field-hint">{t("legacy.aab727ead5bb")}</span>
        </div>
        <div className="field">
          <label>{t("legacy.6e6d6ddbb7c1")}</label>
          <div className="picker-field">
            {draft.members.length > 0 ? (
              draft.members.map((m) => <span key={m} className="tag">{m}</span>)
            ) : (
              <span className="muted" style={{ fontSize: 13 }}>{t("legacy.51c54a24eb22")}</span>
            )}
            <button type="button" className="button" onClick={() => setPicker("members")}>
              <UserPlus size={14} /> {t("legacy.ad9737dafdbf")}
            </button>
          </div>
        </div>
      </Modal>

      <Modal
        open={picker !== null}
        onClose={() => setPicker(null)}
        title={picker === "leader" ? t("legacy.5b894d63abb8") : t("legacy.ad9737dafdbf")}
        subtitle={picker === "leader" ? t("legacy.25a652713caf") : t("legacy.19582c16739e")}
        footer={<button className="button button-primary" onClick={() => setPicker(null)}>{t("legacy.c0b3fbff51cc")}</button>}
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
