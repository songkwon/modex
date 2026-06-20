"use client";

import { useEffect, useState } from "react";
import { Check, Copy, KeyRound, Link2, Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/components/ui/empty-state";
import { Combobox, type ComboOption } from "@/components/ui/combobox";
import {
  createConnectedApp,
  deleteConnectedApp,
  getConnectedApps,
  updateConnectedApp,
  type ConnectedApp,
  type ConnectedAppDraft,
} from "@/lib/api";
import { useI18n } from "@/lib/i18n";

const DEFAULT_SCOPES = ["modex:mcp:read", "modex:docs:read"];

type Draft = ConnectedAppDraft & {
  id?: string;
  redirect_text: string;
};

const emptyDraft: Draft = {
  name: "",
  description: "",
  client_id: "",
  redirect_uris: [],
  scopes: [...DEFAULT_SCOPES],
  trusted: false,
  enabled: true,
  redirect_text: "",
};

function lines(value: string) {
  return value
    .split(/\r?\n|,/)
    .map((v) => v.trim())
    .filter(Boolean);
}

function toDraft(app?: ConnectedApp): Draft {
  if (!app) return emptyDraft;
  return {
    id: app.id,
    name: app.name,
    description: app.description || "",
    client_id: app.client_id,
    redirect_uris: app.redirect_uris || [],
    scopes: app.scopes || [],
    trusted: !!app.trusted,
    enabled: !!app.enabled,
    redirect_text: (app.redirect_uris || []).join("\n"),
  };
}

function payload(draft: Draft): ConnectedAppDraft {
  return {
    name: draft.name.trim(),
    description: draft.description?.trim(),
    redirect_uris: lines(draft.redirect_text),
    scopes: draft.scopes,
    trusted: draft.trusted,
    enabled: draft.enabled,
  };
}

async function copy(text: string, done: (v: boolean) => void) {
  try {
    await navigator.clipboard.writeText(text);
    done(true);
    setTimeout(() => done(false), 1500);
  } catch {
    done(false);
  }
}

export default function AdminConnectedAppsPage() {
  const { t } = useI18n();
  const SCOPE_OPTIONS: ComboOption[] = [
    { value: "modex:mcp:read", label: "modex:mcp:read", hint: t("admin.connectedApps.allow_mcp_tools_to_read_documents") },
    { value: "modex:docs:read", label: "modex:docs:read", hint: t("admin.connectedApps.allow_document_reading_api") },
  ];
  const [apps, setApps] = useState<ConnectedApp[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(emptyDraft);
  const [createdSecret, setCreatedSecret] = useState("");
  const [copied, setCopied] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const res = await getConnectedApps();
      setApps(res.apps || []);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  function openCreate() {
    setDraft(emptyDraft);
    setCreatedSecret("");
    setError("");
    setModalOpen(true);
  }

  function openEdit(app: ConnectedApp) {
    setDraft(toDraft(app));
    setCreatedSecret("");
    setError("");
    setModalOpen(true);
  }

  async function submit() {
    const body = payload(draft);
    if (!body.name || !body.redirect_uris.length) {
      setError(t("admin.connectedApps.name_and_redirect_uri_are_required"));
      return;
    }
    setSaving(true);
    setError("");
    try {
      if (draft.id) {
        await updateConnectedApp(draft.id, body);
        setModalOpen(false);
      } else {
        const created = await createConnectedApp(body);
        setCreatedSecret(created.client_secret || "");
        setDraft(toDraft(created));
      }
      await load();
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  }

  async function remove(app: ConnectedApp) {
    if (!confirm(t("admin.connectedApps.delete_app_value1_existing_authorizations_will_be_revoked", { value1: app.name }))) return;
    setError("");
    try {
      await deleteConnectedApp(app.id);
      await load();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title={t("component.adminShell.app_link")}
      kicker="Connected Apps"
      description={t("admin.connectedApps.description")}
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div>
          <div className="page-kicker">OAuth Applications</div>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            {t("admin.connectedApps.recommended_scope")}<code className="code-chip">modex:mcp:read</code>、<code className="code-chip">modex:docs:read</code>
          </p>
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> {t("admin.connectedApps.create_application")}
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("admin.connectedApps.apply")}</th>
                <th>Client ID</th>
                <th>Scopes</th>
                <th>{t("admin.releases.status")}</th>
                <th>{t("admin.connectedApps.recently_used")}</th>
                <th className="table-actions-col">{t("admin.modules.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {apps.map((app) => (
                <tr key={app.id}>
                  <td>
                    <div style={{ fontWeight: 640 }}>{app.name}</div>
                    {app.description ? <div className="muted" style={{ fontSize: 12 }}>{app.description}</div> : null}
                  </td>
                  <td><code className="code-chip">{app.client_id}</code></td>
                  <td>
                    <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                      {(app.scopes || []).map((scope) => <span key={scope} className="badge">{scope}</span>)}
                    </div>
                  </td>
                  <td>
                    <span className={`badge ${app.enabled ? "badge-success" : "badge-danger"}`}>
                      {app.enabled ? t("admin.connectedApps.enable") : t("admin.connectedApps.deactivate")}
                    </span>
                    {app.trusted ? <span className="badge" style={{ marginLeft: 4 }}>Trusted</span> : null}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {app.last_used_at ? new Date(app.last_used_at).toLocaleString() : "—"}
                  </td>
                  <td className="table-actions-cell">
                    <div className="row-actions">
                      <button className="icon-btn" onClick={() => openEdit(app)} aria-label={t("admin.categories.edit")}><Pencil size={14} /></button>
                      <button className="icon-btn" onClick={() => remove(app)} aria-label={t("admin.categories.delete")}><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
              {!apps.length && !loading ? (
                <tr>
                  <td colSpan={6}>
                    <EmptyState icon={Link2} title={t("admin.connectedApps.no_app_links_yet")} hint={t("admin.connectedApps.create_a_new_oauth_application_to_allow_external")} />
                  </td>
                </tr>
              ) : null}
              {loading ? (
                <tr>
                  <td colSpan={6}><span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><Loader2 size={15} className="ds-spin" /> {t("component.docReadStats.loading")}</span></td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={draft.id ? t("admin.connectedApps.edit_app_value1", { value1: draft.name }) : t("admin.connectedApps.create_application_link")}
        subtitle={t("admin.connectedApps.the_client_secret_is_displayed_only_once_upon")}
        width={720}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{createdSecret ? t("admin.modules.done") : t("admin.categories.cancel")}</button>
            <button className="button button-primary" onClick={submit} disabled={saving || !draft.name.trim() || !draft.redirect_text.trim()}>
              {saving ? <Loader2 size={16} className="ds-spin" /> : <KeyRound size={16} />} {draft.id ? t("admin.modules.save") : t("admin.categories.create")}
            </button>
          </>
        }
      >
        {createdSecret ? (
          <div className="field" style={{ border: "1px solid hsl(var(--accent-border))", borderRadius: 14, padding: 12, background: "hsl(var(--accent-soft))" }}>
            <label>{t("admin.connectedApps.client_secret_shown_only_once")}</label>
            <div style={{ display: "flex", gap: 8 }}>
              <input readOnly value={createdSecret} style={{ fontFamily: "ui-monospace, monospace", fontSize: 12 }} />
              <button className="button" onClick={() => copy(createdSecret, (v) => setCopied(v ? "secret" : ""))}>
                {copied === "secret" ? <Check size={14} /> : <Copy size={14} />}
              </button>
            </div>
            <span className="field-hint">{t("admin.connectedApps.save_this_securely_in_your_external_application_immediately")}</span>
          </div>
        ) : null}

        <div className="field">
          <label>{t("admin.connectedApps.app_name")}</label>
          <input value={draft.name} autoFocus placeholder={t("admin.connectedApps.e_g_internal_mcp_gateway")} onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          <span className="field-hint">{t("admin.connectedApps.client_id_is_auto_generated_and_displayed_in")}</span>
        </div>

        {draft.id ? (
          <div className="field">
            <label>Client ID</label>
            <input value={draft.client_id || ""} readOnly style={{ fontFamily: "ui-monospace, monospace", fontSize: 12 }} />
            <span className="field-hint">{t("admin.connectedApps.client_id_cannot_be_modified_after_app_creation")}</span>
          </div>
        ) : null}

        <div className="field">
          <label>{t("admin.categories.description")}</label>
          <input value={draft.description || ""} placeholder={t("admin.connectedApps.describe_how_this_app_accesses_modex")} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>

        <div className="field">
          <label>Redirect URI *</label>
          <textarea
            value={draft.redirect_text}
            placeholder="https://example.com/oauth/modex/callback"
            style={{ minHeight: 84, fontFamily: "ui-monospace, monospace", fontSize: 13 }}
            onChange={(e) => setDraft({ ...draft, redirect_text: e.target.value })}
          />
          <span className="field-hint">{t("admin.connectedApps.one_uri_per_line_exact_match_required_for")}</span>
        </div>

        <div className="field">
          <label>Scopes</label>
          <Combobox
            options={SCOPE_OPTIONS}
            value={draft.scopes}
            onChange={(scopes) => setDraft({ ...draft, scopes })}
            placeholder={t("admin.connectedApps.select_authorization_scope")}
          />
          <span className="field-hint">{t("admin.connectedApps.mcp_integration_should_include_at_least_modex_mcp")}</span>
        </div>

        <div className="field-row">
          <Switch
            checked={draft.enabled}
            onChange={(enabled) => setDraft({ ...draft, enabled })}
            label={t("admin.connectedApps.enable_app")}
            hint={t("admin.connectedApps.after_closing_token_exchange_and_bearer_access_will")}
          />
          <Switch
            checked={draft.trusted}
            onChange={(trusted) => setDraft({ ...draft, trusted })}
            label="Trusted"
            hint={t("admin.connectedApps.trusted_apps_can_skip_the_user_authorization_confirmation")}
          />
        </div>
      </Modal>
    </AdminShell>
  );
}
