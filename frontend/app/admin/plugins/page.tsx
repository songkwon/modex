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
    diagram: t("admin.plugins.chart"),
    math: t("admin.plugins.math"),
    content: t("admin.plugins.content_enhancement"),
    api: "API",
    custom: t("admin.plugins.customize")
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
    if (!confirm(t("admin.plugins.delete_import_plugin_value1", { value1: key }))) return;
    try {
      const r = await deletePlugin(key);
      setPlugins(r.plugins || []);
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title={t("component.adminShell.plugin_management")}
      kicker="Plugins"
      description={t("admin.plugins.enable_disable_built_in_capabilities_and_configure_parameters")}
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
          <h2 className="page-kicker" style={{ margin: 0 }}>{t("admin.plugins.plugin_imported")}</h2>
          <button className="button" onClick={() => { setDraft(EMPTY_DRAFT); setImportErr(""); setShowImport(true); }}>
            <Upload size={15} /> {t("admin.plugins.import_plugin")}
          </button>
        </div>
        {uploaded.length === 0 ? <p className="muted" style={{ fontSize: 13 }}>{t("admin.plugins.no_imported_plugins_click_import_in_the_top")}</p> : null}
        {uploaded.map((p) => (
          <div key={p.key} className="plugin-row">
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <label className="plugin-toggle">
                <input type="checkbox" checked={p.enabled} onChange={(e) => setEnabled(p.key, e.target.checked)} />
                <span className="plugin-row__name">{p.name}</span>
              </label>
              <span className="tag">{p.kind === "fence" ? `\`\`\`${p.lang}` : `<${p.tag}/>`}</span>
              <button className="icon-btn" style={{ marginLeft: "auto" }} aria-label={t("admin.categories.delete")} onClick={() => remove(p.key)}>
                <Trash2 size={14} />
              </button>
            </div>
            {p.description ? <p className="plugin-row__desc">{p.description}</p> : null}
          </div>
        ))}
      </section>

      <details className="card">
        <summary style={{ cursor: "pointer", fontWeight: 600 }}>{t("admin.plugins.development_methods_how_to_write_plugins")}</summary>
        <div className="mdx" style={{ marginTop: 12, fontSize: 13.5, lineHeight: 1.7 }}>
          <p>{t("admin.plugins.plugin_usage")} <b>{t("admin.plugins.jsx_react_source_code")}</b> {t("admin.plugins.write_run_on")}<b>{t("admin.plugins.isolated_sandbox_iframe")}</b> {t("admin.plugins.in_cannot_read_this_site_s_cookies_login")} <code>Plugin</code> {t("admin.plugins.components")}</p>
          <ul>
            <li><b>{t("admin.plugins.component_plugin_component")}</b>{t("admin.plugins.displayed_in_documentation_as")} <code>{"<Tag .../>"}</code> {t("admin.plugins.use_tag_attribute_as")} <code>props</code> {t("admin.plugins.to_be_passed_in_string_number_boolean_properties")}</li>
            <li><b>{t("admin.plugins.fence_plugin")}</b>{t("admin.plugins.intercepts_a_specific_code_fence_language_fence_content")} <code>props.source</code> {t("admin.plugins.to_be_passed_in")}</li>
          </ul>
          <p>{t("admin.plugins.component_example")}</p>
          <pre className="mdx-code__pre" style={{ whiteSpace: "pre-wrap" }}>{EXAMPLE_COMPONENT}</pre>
          <p>{t("admin.plugins.fence_example")}</p>
          <pre className="mdx-code__pre" style={{ whiteSpace: "pre-wrap" }}>{EXAMPLE_FENCE}</pre>
          <p className="muted" style={{ fontSize: 12.5 }}>
            {t("admin.plugins.security_notice_the_sandbox_prevents_reading_this_site")}<b>{t("component.searchResults.close")}</b>{t("admin.plugins.must_be_manually_enabled_global_access_is_available")} <code>React</code> / <code>ReactDOM</code>。
          </p>
        </div>
      </details>

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button className="button button-primary" onClick={submit} disabled={saving || !plugins.length}>
          {saving ? <Loader2 size={16} className="ds-spin" /> : <Plug size={16} />} {t("admin.modules.save")}
        </button>
        {saved ? <span className="badge badge-success"><Check size={13} /> {t("admin.snippets.saved")}</span> : null}
      </div>

      <Modal
        open={showImport}
        title={t("admin.plugins.import_plugin")}
        subtitle={t("admin.plugins.jsx_source_code_runs_in_an_isolated_sandbox")}
        width={640}
        onClose={() => setShowImport(false)}
        footer={
          <>
            <button className="button" onClick={() => setShowImport(false)}>{t("admin.categories.cancel")}</button>
            <button className="button button-primary" onClick={doImport} disabled={importing}>
              {importing ? <Loader2 size={15} className="ds-spin" /> : <Plus size={15} />} {t("admin.plugins.import_disabled_by_default")}
            </button>
          </>
        }
      >
        {importErr ? <div className="panel badge-danger" style={{ borderRadius: 10, marginBottom: 10 }}>{importErr}</div> : null}
        <div style={{ display: "grid", gap: 12 }}>
              <div style={{ display: "flex", gap: 10 }}>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("admin.plugins.key_lowercase_hyphenated")}</label>
                  <input value={draft.key} placeholder="my-plugin" onChange={(e) => setDraft({ ...draft, key: e.target.value })} />
                </div>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("admin.plugins.name")}</label>
                  <input value={draft.name} placeholder={t("admin.plugins.my_plugins")} onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
                </div>
              </div>
              <div className="field">
                <label>{t("admin.plugins.description_optional")}</label>
                <input value={draft.description} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
              </div>
              <div style={{ display: "flex", gap: 10 }}>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("admin.plugins.type")}</label>
                  <select value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value as Draft["kind"] })}>
                    <option value="component">{t("admin.plugins.components_tag")}</option>
                    <option value="fence">围栏（```lang）</option>
                  </select>
                </div>
                <div className="field" style={{ flex: 1 }}>
                  <label>{t("common.category")}</label>
                  <select value={draft.category} onChange={(e) => setDraft({ ...draft, category: e.target.value })}>
                    {Object.entries(CATEGORY_LABELS).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                  </select>
                </div>
                <div className="field" style={{ flex: 1 }}>
                  {draft.kind === "component" ? (
                    <>
                      <label>{t("admin.plugins.tag_name_capitalized")}</label>
                      <input value={draft.tag} placeholder="MyTag" onChange={(e) => setDraft({ ...draft, tag: e.target.value })} />
                    </>
                  ) : (
                    <>
                      <label>{t("admin.plugins.fence_language_lowercase")}</label>
                      <input value={draft.lang} placeholder="mylang" onChange={(e) => setDraft({ ...draft, lang: e.target.value })} />
                    </>
                  )}
                </div>
              </div>
              <div className="field">
                <label>{t("admin.plugins.jsx_source_code_must_define")} <code>function Plugin(props)</code>）</label>
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
