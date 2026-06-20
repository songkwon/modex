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
    { value: "modex:mcp:read", label: "modex:mcp:read", hint: t("legacy.17d06d4ffd09") },
    { value: "modex:docs:read", label: "modex:docs:read", hint: t("legacy.9c0fa2f3543e") },
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
      setError(t("legacy.c73a818cda52"));
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
    if (!confirm(t("legacy.11d1cfc0b20f", { value1: app.name }))) return;
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
      title={t("legacy.213f5505bd1a")}
      kicker="Connected Apps"
      description={t("legacy.7d008287b514")}
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div>
          <div className="page-kicker">OAuth Applications</div>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            {t("legacy.900e9dc7b94b")}<code className="code-chip">modex:mcp:read</code>、<code className="code-chip">modex:docs:read</code>
          </p>
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> {t("legacy.8974b8d346cb")}
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t("legacy.63c73c4730f4")}</th>
                <th>Client ID</th>
                <th>Scopes</th>
                <th>{t("legacy.6320b4a8722a")}</th>
                <th>{t("legacy.39a9046f9c8c")}</th>
                <th style={{ textAlign: "right" }}>{t("legacy.ed31fbb483ee")}</th>
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
                      {app.enabled ? t("legacy.f4f0ead1116b") : t("legacy.4e6fd0e28c55")}
                    </span>
                    {app.trusted ? <span className="badge" style={{ marginLeft: 4 }}>Trusted</span> : null}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {app.last_used_at ? new Date(app.last_used_at).toLocaleString() : "—"}
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="icon-btn" onClick={() => openEdit(app)} aria-label={t("legacy.051836569928")}><Pencil size={14} /></button>
                      <button className="icon-btn" onClick={() => remove(app)} aria-label={t("legacy.2f9daa828907")}><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
              {!apps.length && !loading ? (
                <tr>
                  <td colSpan={6}>
                    <EmptyState icon={Link2} title={t("legacy.f225c5556445")} hint={t("legacy.ddaf636a33c1")} />
                  </td>
                </tr>
              ) : null}
              {loading ? (
                <tr>
                  <td colSpan={6}><span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><Loader2 size={15} className="ds-spin" /> {t("legacy.4927a53bcc88")}</span></td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      <section className="card" style={{ display: "grid", gap: 10 }}>
        <div className="page-kicker">OAuth endpoints</div>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
          <code className="code-chip">/.well-known/oauth-authorization-server</code>
          <code className="code-chip">/oauth/authorize</code>
          <code className="code-chip">/oauth/token</code>
          <code className="code-chip">/oauth/revoke</code>
        </div>
      </section>

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={draft.id ? t("legacy.c624c6ca40d9", { value1: draft.name }) : t("legacy.ce99c2ee524c")}
        subtitle={t("legacy.cee6703ce528")}
        width={720}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{createdSecret ? t("legacy.c0b3fbff51cc") : t("legacy.2cd0f3be8738")}</button>
            <button className="button button-primary" onClick={submit} disabled={saving || !draft.name.trim() || !draft.redirect_text.trim()}>
              {saving ? <Loader2 size={16} className="ds-spin" /> : <KeyRound size={16} />} {draft.id ? t("legacy.a3030bf8f16d") : t("legacy.cde2cd071d25")}
            </button>
          </>
        }
      >
        {createdSecret ? (
          <div className="field" style={{ border: "1px solid hsl(var(--accent-border))", borderRadius: 14, padding: 12, background: "hsl(var(--accent-soft))" }}>
            <label>{t("legacy.1f810545799c")}</label>
            <div style={{ display: "flex", gap: 8 }}>
              <input readOnly value={createdSecret} style={{ fontFamily: "ui-monospace, monospace", fontSize: 12 }} />
              <button className="button" onClick={() => copy(createdSecret, (v) => setCopied(v ? "secret" : ""))}>
                {copied === "secret" ? <Check size={14} /> : <Copy size={14} />}
              </button>
            </div>
            <span className="field-hint">{t("legacy.a7a4d24d4e85")}</span>
          </div>
        ) : null}

        <div className="field">
          <label>{t("legacy.c871bddf3633")}</label>
          <input value={draft.name} autoFocus placeholder={t("legacy.4bc9b4d2219a")} onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          <span className="field-hint">{t("legacy.445bae65f5fa")}</span>
        </div>

        {draft.id ? (
          <div className="field">
            <label>Client ID</label>
            <input value={draft.client_id || ""} readOnly style={{ fontFamily: "ui-monospace, monospace", fontSize: 12 }} />
            <span className="field-hint">{t("legacy.3232f160da88")}</span>
          </div>
        ) : null}

        <div className="field">
          <label>{t("legacy.dc2ba467fc7a")}</label>
          <input value={draft.description || ""} placeholder={t("legacy.a428f7c3d4b5")} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>

        <div className="field">
          <label>Redirect URI *</label>
          <textarea
            value={draft.redirect_text}
            placeholder="https://example.com/oauth/modex/callback"
            style={{ minHeight: 84, fontFamily: "ui-monospace, monospace", fontSize: 13 }}
            onChange={(e) => setDraft({ ...draft, redirect_text: e.target.value })}
          />
          <span className="field-hint">{t("legacy.007dcf4c2a4c")}</span>
        </div>

        <div className="field">
          <label>Scopes</label>
          <Combobox
            options={SCOPE_OPTIONS}
            value={draft.scopes}
            onChange={(scopes) => setDraft({ ...draft, scopes })}
            placeholder={t("legacy.862aa2cfad14")}
          />
          <span className="field-hint">{t("legacy.af793bf18819")}</span>
        </div>

        <div className="field-row">
          <Switch
            checked={draft.enabled}
            onChange={(enabled) => setDraft({ ...draft, enabled })}
            label={t("legacy.aa50aae6ada8")}
            hint={t("legacy.e76e77cb411f")}
          />
          <Switch
            checked={draft.trusted}
            onChange={(trusted) => setDraft({ ...draft, trusted })}
            label="Trusted"
            hint={t("legacy.b3bf0cf5def6")}
          />
        </div>
      </Modal>
    </AdminShell>
  );
}
