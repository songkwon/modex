"use client";

import { useEffect, useMemo, useState } from "react";
import { Check, Loader2, Plug, Plus, Trash2, Upload } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Modal } from "@/components/ui/modal";
import {
  getPlugins,
  savePlugins,
  importPlugin,
  deletePlugin,
  type PluginState,
  type PluginConfig,
  type UploadedPlugin
} from "@/lib/api";
import { useI18n } from "@/lib/i18n";


const EXAMPLE_COMPONENT = `// 组件插件：在文档里写 <Hello name="世界" />
// 必须定义一个名为 Plugin 的组件，props 即标签属性。
function Plugin({ name }) {
  return <button style={{ padding: "8px 14px", borderRadius: 8 }}>
    你好，{name || "世界"}
  </button>;
}`;

const EXAMPLE_FENCE = `// 围栏插件：拦截 \`\`\`echo 代码块，源码通过 props.source 传入。
function Plugin({ source }) {
  return <pre style={{ background: "#fdf6e3", padding: 10 }}>{source}</pre>;
}`;

type Draft = {
  key: string;
  name: string;
  description: string;
  category: string;
  kind: "component" | "fence";
  tag: string;
  lang: string;
  code: string;
};

const EMPTY_DRAFT: Draft = { key: "", name: "", description: "", category: "custom", kind: "component", tag: "", lang: "", code: EXAMPLE_COMPONENT };

export default function AdminPluginsPage() {
  const { t } = useI18n();
  const CATEGORY_LABELS: Record<string, string> = {
    diagram: t("legacy.8cb443ab8379"),
    math: t("legacy.872c1fa141b5"),
    content: t("legacy.6bd50e1c9c95"),
    api: "API",
    custom: t("legacy.4eafa9e925b3")
  };
  const [plugins, setPlugins] = useState<PluginState[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const [showImport, setShowImport] = useState(false);
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [importing, setImporting] = useState(false);
  const [importErr, setImportErr] = useState("");

  useEffect(() => {
    getPlugins()
      .then((r) => setPlugins(r.plugins || []))
      .catch((e) => setError(String(e)));
  }, []);

  const builtinGroups = useMemo(() => {
    const order = ["diagram", "math", "content", "api"];
    const by: Record<string, PluginState[]> = {};
    for (const p of plugins) if (!p.uploaded) (by[p.category] ||= []).push(p);
    return order.filter((c) => by[c]?.length).map((c) => [c, by[c]] as const);
  }, [plugins]);
  const uploaded = useMemo(() => plugins.filter((p) => p.uploaded), [plugins]);

  function setEnabled(key: string, enabled: boolean) {
    setPlugins((ps) => ps.map((p) => (p.key === key ? { ...p, enabled } : p)));
  }
  function setConfig(key: string, field: string, value: string) {
    setPlugins((ps) => ps.map((p) => (p.key === key ? { ...p, config: { ...p.config, [field]: value } } : p)));
  }

  async function submit() {
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      const payload: PluginConfig = {};
      for (const p of plugins) payload[p.key] = { enabled: p.enabled, config: p.config };
      const r = await savePlugins(payload);
      setPlugins(r.plugins || []);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  }

  async function doImport() {
    setImporting(true);
    setImportErr("");
    try {
      const body: UploadedPlugin = {
        key: draft.key.trim(),
        name: draft.name.trim(),
        description: draft.description.trim(),
        category: draft.category,
        kind: draft.kind,
        tag: draft.kind === "component" ? draft.tag.trim() : undefined,
        lang: draft.kind === "fence" ? draft.lang.trim() : undefined,
        code: draft.code
      };
      const r = await importPlugin(body);
      setPlugins(r.plugins || []);
      setShowImport(false);
      setDraft(EMPTY_DRAFT);
    } catch (e) {
      setImportErr(String(e));
    } finally {
      setImporting(false);
    }
  }

  async function remove(key: string) {
    if (!confirm(t("legacy.8a36ff358006", { value1: key }))) return;
    try {
      const r = await deletePlugin(key);
      setPlugins(r.plugins || []);
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title={t("legacy.35fcbd57d58a")}
      kicker="Plugins"
      description={t("legacy.acaae5c0fee0")}
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      {builtinGroups.map(([category, items]) => (
        <section key={category} className="card" style={{ display: "grid", gap: 14 }}>
          <h2 className="page-kicker" style={{ margin: 0 }}>{CATEGORY_LABELS[category] || category}</h2>
          {items.map((p) => (
            <div key={p.key} className="plugin-row">
              <label className="plugin-toggle">
                <input type="checkbox" checked={p.enabled} onChange={(e) => setEnabled(p.key, e.target.checked)} />
                <span className="plugin-row__name">{p.name}</span>
              </label>
              <p className="plugin-row__desc">{p.description}</p>
              {p.enabled && p.fields?.length ? (
                <div className="plugin-row__fields">
                  {p.fields.map((f) => (
                    <div className="field" key={f.key}>
                      <label>{f.label}</label>
                      <input
                        value={p.config?.[f.key] || ""}
                        placeholder={f.placeholder || ""}
                        onChange={(e) => setConfig(p.key, f.key, e.target.value)}
                      />
                    </div>
                  ))}
                </div>
              ) : null}
            </div>
          ))}
        </section>
      ))}

      <section className="card" style={{ display: "grid", gap: 14 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h2 className="page-kicker" style={{ margin: 0 }}>{t("legacy.48d95f8c1652")}</h2>
          <button className="button" onClick={() => { setDraft(EMPTY_DRAFT); setImportErr(""); setShowImport(true); }}>
            <Upload size={15} /> {t("legacy.7a1b0c1add81")}
          </button>
        </div>
        {uploaded.length === 0 ? <p className="muted" style={{ fontSize: 13 }}>{t("legacy.ad69978c3b62")}</p> : null}
        {uploaded.map((p) => (
          <div key={p.key} className="plugin-row">
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <label className="plugin-toggle">
                <input type="checkbox" checked={p.enabled} onChange={(e) => setEnabled(p.key, e.target.checked)} />
                <span className="plugin-row__name">{p.name}</span>
              </label>
              <span className="tag">{p.kind === "fence" ? `\`\`\`${p.lang}` : `<${p.tag}/>`}</span>
              <button className="icon-btn" style={{ marginLeft: "auto" }} aria-label={t("legacy.2f9daa828907")} onClick={() => remove(p.key)}>
                <Trash2 size={14} />
              </button>
            </div>
            {p.description ? <p className="plugin-row__desc">{p.description}</p> : null}
          </div>
        ))}
      </section>

      <details className="card">
        <summary style={{ cursor: "pointer", fontWeight: 600 }}>{t("legacy.7779ad4aa520")}</summary>
        <div className="mdx" style={{ marginTop: 12, fontSize: 13.5, lineHeight: 1.7 }}>
          <p>{t("legacy.9b9f621fd3a4")} <b>{t("legacy.81fa5829f91e")}</b> {t("legacy.46a153734cab")}<b>{t("legacy.5e26fd1660c7")}</b> {t("legacy.fa701a95f1dd")} <code>Plugin</code> {t("legacy.12851b6a3a6d")}</p>
          <ul>
            <li><b>{t("legacy.b0132357e30d")}</b>{t("legacy.a0bbaf0c2187")} <code>{"<Tag .../>"}</code> {t("legacy.6bcda8d93d53")} <code>props</code> {t("legacy.66148c036139")}</li>
            <li><b>{t("legacy.53345ce44995")}</b>{t("legacy.ccd097a2d258")} <code>props.source</code> {t("legacy.fdf954d50113")}</li>
          </ul>
          <p>{t("legacy.1826d6798e19")}</p>
          <pre className="mdx-code__pre" style={{ whiteSpace: "pre-wrap" }}>{EXAMPLE_COMPONENT}</pre>
          <p>{t("legacy.2ed3737a5644")}</p>
          <pre className="mdx-code__pre" style={{ whiteSpace: "pre-wrap" }}>{EXAMPLE_FENCE}</pre>
          <p className="muted" style={{ fontSize: 12.5 }}>
            {t("legacy.fa08fffa7112")}<b>{t("legacy.3fd47edce45b")}</b>{t("legacy.d710928db6f9")} <code>React</code> / <code>ReactDOM</code>。
          </p>
        </div>
      </details>

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button className="button button-primary" onClick={submit} disabled={saving || !plugins.length}>
          {saving ? <Loader2 size={16} className="ds-spin" /> : <Plug size={16} />} {t("legacy.a3030bf8f16d")}
        </button>
        {saved ? <span className="badge badge-success"><Check size={13} /> {t("legacy.1bd91a7d0c53")}</span> : null}
      </div>

      <Modal
        open={showImport}
        title={t("legacy.7a1b0c1add81")}
        subtitle={t("legacy.ed32b43a4d6f")}
        width={640}
        onClose={() => setShowImport(false)}
        footer={
          <>
            <button className="button" onClick={() => setShowImport(false)}>{t("legacy.2cd0f3be8738")}</button>
            <button className="button button-primary" onClick={doImport} disabled={importing}>
              {importing ? <Loader2 size={15} className="ds-spin" /> : <Plus size={15} />} {t("legacy.f805e562d128")}
            </button>
          </>
        }
      >
        {importErr ? <div className="panel badge-danger" style={{ borderRadius: 10, marginBottom: 10 }}>{importErr}</div> : null}
        <div style={{ display: "grid", gap: 12 }}>
              <div style={{ display: "flex", gap: 10 }}>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("legacy.1c69bd706188")}</label>
                  <input value={draft.key} placeholder="my-plugin" onChange={(e) => setDraft({ ...draft, key: e.target.value })} />
                </div>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("legacy.d44e9b3d3b31")}</label>
                  <input value={draft.name} placeholder={t("legacy.1b70686b644a")} onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
                </div>
              </div>
              <div className="field">
                <label>{t("legacy.bd5b18f0c139")}</label>
                <input value={draft.description} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
              </div>
              <div style={{ display: "flex", gap: 10 }}>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("legacy.ba40014ff496")}</label>
                  <select value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value as Draft["kind"] })}>
                    <option value="component">{t("legacy.1db0a5ae46ee")}</option>
                    <option value="fence">围栏（```lang）</option>
                  </select>
                </div>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("legacy.515559957fd3")}</label>
                  <select value={draft.category} onChange={(e) => setDraft({ ...draft, category: e.target.value })}>
                    {Object.entries(CATEGORY_LABELS).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                  </select>
                </div>
                <div className="field" style={{ flex: 1 }}>
                  {draft.kind === "component" ? (
                    <>
                      <label>{t("legacy.c4c09dfac632")}</label>
                      <input value={draft.tag} placeholder="MyTag" onChange={(e) => setDraft({ ...draft, tag: e.target.value })} />
                    </>
                  ) : (
                    <>
                      <label>{t("legacy.8edb428a298f")}</label>
                      <input value={draft.lang} placeholder="mylang" onChange={(e) => setDraft({ ...draft, lang: e.target.value })} />
                    </>
                  )}
                </div>
              </div>
              <div className="field">
                <label>{t("legacy.ba6efad1547d")} <code>function Plugin(props)</code>）</label>
                <textarea
                  style={{ minHeight: 180, fontFamily: "ui-monospace, monospace", fontSize: 12.5, padding: "10px 12px", lineHeight: 1.6 }}
                  value={draft.code}
                  onChange={(e) => setDraft({ ...draft, code: e.target.value })}
                />
              </div>
            </div>
      </Modal>
    </AdminShell>
  );
}
