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

const DEFAULT_SCOPES = ["modex:mcp:read", "modex:docs:read"];
const SCOPE_OPTIONS: ComboOption[] = [
  { value: "modex:mcp:read", label: "modex:mcp:read", hint: "允许 MCP 工具读取文档" },
  { value: "modex:docs:read", label: "modex:docs:read", hint: "允许读取文档 API" },
];

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
      setError("名称和 Redirect URI 必填。");
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
    if (!confirm(`删除应用「${app.name}」？已有授权会被撤销。`)) return;
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
      title="应用链接"
      kicker="Connected Apps"
      description="注册外部应用，让它们通过 OAuth 授权访问 Modex API 与 MCP。Modex 保持通用开源能力，内部网关只是这里的一个普通应用。"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <div className="admin-toolbar">
        <div>
          <div className="page-kicker">OAuth Applications</div>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            推荐 scope：<code className="code-chip">modex:mcp:read</code>、<code className="code-chip">modex:docs:read</code>
          </p>
        </div>
        <div className="admin-toolbar-actions">
          <button className="button button-primary" onClick={openCreate}>
            <Plus size={16} /> 新建应用
          </button>
        </div>
      </div>

      <div className="table-card">
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>应用</th>
                <th>Client ID</th>
                <th>Scopes</th>
                <th>状态</th>
                <th>最近使用</th>
                <th style={{ textAlign: "right" }}>操作</th>
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
                      {app.enabled ? "启用" : "停用"}
                    </span>
                    {app.trusted ? <span className="badge" style={{ marginLeft: 4 }}>Trusted</span> : null}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {app.last_used_at ? new Date(app.last_used_at).toLocaleString() : "—"}
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="icon-btn" onClick={() => openEdit(app)} aria-label="编辑"><Pencil size={14} /></button>
                      <button className="icon-btn" onClick={() => remove(app)} aria-label="删除"><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
              {!apps.length && !loading ? (
                <tr>
                  <td colSpan={6}>
                    <EmptyState icon={Link2} title="还没有应用链接" hint="新建一个 OAuth 应用，外部系统即可通过授权码流程接入 Modex。" />
                  </td>
                </tr>
              ) : null}
              {loading ? (
                <tr>
                  <td colSpan={6}><span className="muted" style={{ display: "inline-flex", alignItems: "center", gap: 8 }}><Loader2 size={15} className="ds-spin" /> 加载中…</span></td>
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
        title={draft.id ? `编辑应用 · ${draft.name}` : "新建应用链接"}
        subtitle="Client Secret 只会在创建成功后显示一次"
        width={720}
        footer={
          <>
            <button className="button" onClick={() => setModalOpen(false)}>{createdSecret ? "完成" : "取消"}</button>
            <button className="button button-primary" onClick={submit} disabled={saving || !draft.name.trim() || !draft.redirect_text.trim()}>
              {saving ? <Loader2 size={16} className="ds-spin" /> : <KeyRound size={16} />} {draft.id ? "保存" : "创建"}
            </button>
          </>
        }
      >
        {createdSecret ? (
          <div className="field" style={{ border: "1px solid hsl(var(--accent-border))", borderRadius: 14, padding: 12, background: "hsl(var(--accent-soft))" }}>
            <label>Client Secret（仅显示一次）</label>
            <div style={{ display: "flex", gap: 8 }}>
              <input readOnly value={createdSecret} style={{ fontFamily: "ui-monospace, monospace", fontSize: 12 }} />
              <button className="button" onClick={() => copy(createdSecret, (v) => setCopied(v ? "secret" : ""))}>
                {copied === "secret" ? <Check size={14} /> : <Copy size={14} />}
              </button>
            </div>
            <span className="field-hint">请立即保存到外部应用的安全配置中，关闭弹窗后无法再次查看。</span>
          </div>
        ) : null}

        <div className="field">
          <label>应用名称 *</label>
          <input value={draft.name} autoFocus placeholder="如 Internal MCP Gateway" onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
          <span className="field-hint">Client ID 会自动生成，创建后在列表和编辑弹窗里展示。</span>
        </div>

        {draft.id ? (
          <div className="field">
            <label>Client ID</label>
            <input value={draft.client_id || ""} readOnly style={{ fontFamily: "ui-monospace, monospace", fontSize: 12 }} />
            <span className="field-hint">已创建应用不可修改 Client ID。</span>
          </div>
        ) : null}

        <div className="field">
          <label>描述</label>
          <input value={draft.description || ""} placeholder="说明这个应用会如何访问 Modex" onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
        </div>

        <div className="field">
          <label>Redirect URI *</label>
          <textarea
            value={draft.redirect_text}
            placeholder="https://example.com/oauth/modex/callback"
            style={{ minHeight: 84, fontFamily: "ui-monospace, monospace", fontSize: 13 }}
            onChange={(e) => setDraft({ ...draft, redirect_text: e.target.value })}
          />
          <span className="field-hint">每行一个 URI，授权时必须完全匹配。</span>
        </div>

        <div className="field">
          <label>Scopes</label>
          <Combobox
            options={SCOPE_OPTIONS}
            value={draft.scopes}
            onChange={(scopes) => setDraft({ ...draft, scopes })}
            placeholder="选择授权范围…"
          />
          <span className="field-hint">MCP 集成建议至少包含 modex:mcp:read。</span>
        </div>

        <div className="field-row">
          <Switch
            checked={draft.enabled}
            onChange={(enabled) => setDraft({ ...draft, enabled })}
            label="启用应用"
            hint="关闭后 token 兑换和 bearer 访问都会失效。"
          />
          <Switch
            checked={draft.trusted}
            onChange={(trusted) => setDraft({ ...draft, trusted })}
            label="Trusted"
            hint="受信任应用可跳过用户授权确认页。"
          />
        </div>
      </Modal>
    </AdminShell>
  );
}
