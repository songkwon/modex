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

const CATEGORY_LABELS: Record<string, string> = {
  diagram: "图表",
  math: "数学",
  content: "内容增强",
  api: "API",
  custom: "自定义"
};

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
    if (!confirm(`删除导入插件「${key}」？`)) return;
    try {
      const r = await deletePlugin(key);
      setPlugins(r.plugins || []);
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <AdminShell
      title="插件管理"
      kicker="Plugins"
      description="开启/关闭内置能力并配置参数，或导入自定义插件。改动保存后立即对全站文档生效。"
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
          <h2 className="page-kicker" style={{ margin: 0 }}>已导入插件</h2>
          <button className="button" onClick={() => { setDraft(EMPTY_DRAFT); setImportErr(""); setShowImport(true); }}>
            <Upload size={15} /> 导入插件
          </button>
        </div>
        {uploaded.length === 0 ? <p className="muted" style={{ fontSize: 13 }}>暂无导入插件。点击右上角导入。</p> : null}
        {uploaded.map((p) => (
          <div key={p.key} className="plugin-row">
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <label className="plugin-toggle">
                <input type="checkbox" checked={p.enabled} onChange={(e) => setEnabled(p.key, e.target.checked)} />
                <span className="plugin-row__name">{p.name}</span>
              </label>
              <span className="tag">{p.kind === "fence" ? `\`\`\`${p.lang}` : `<${p.tag}/>`}</span>
              <button className="icon-btn" style={{ marginLeft: "auto" }} aria-label="删除" onClick={() => remove(p.key)}>
                <Trash2 size={14} />
              </button>
            </div>
            {p.description ? <p className="plugin-row__desc">{p.description}</p> : null}
          </div>
        ))}
      </section>

      <details className="card">
        <summary style={{ cursor: "pointer", fontWeight: 600 }}>开发方法 / 如何编写插件</summary>
        <div className="mdx" style={{ marginTop: 12, fontSize: 13.5, lineHeight: 1.7 }}>
          <p>插件用 <b>JSX/React 源码</b> 编写，运行在<b>隔离的沙箱 iframe</b> 中（无法读取本站 Cookie / 登录态 / DOM）。代码必须定义一个名为 <code>Plugin</code> 的组件：</p>
          <ul>
            <li><b>组件插件（component）</b>：在文档里以 <code>{"<Tag .../>"}</code> 使用，标签属性作为 <code>props</code> 传入（仅支持字符串/数字/布尔属性）。</li>
            <li><b>围栏插件（fence）</b>：拦截某种代码围栏语言，围栏内容作为 <code>props.source</code> 传入。</li>
          </ul>
          <p>组件示例：</p>
          <pre className="mdx-code__pre" style={{ whiteSpace: "pre-wrap" }}>{EXAMPLE_COMPONENT}</pre>
          <p>围栏示例：</p>
          <pre className="mdx-code__pre" style={{ whiteSpace: "pre-wrap" }}>{EXAMPLE_FENCE}</pre>
          <p className="muted" style={{ fontSize: 12.5 }}>
            安全说明：沙箱可阻止读取本站登录态，但插件仍可发起网络请求、展示任意 UI、消耗资源。请仅导入可信来源的代码（导入后默认<b>关闭</b>，需手动开启）。沙箱内可用全局 <code>React</code> / <code>ReactDOM</code>。
          </p>
        </div>
      </details>

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button className="button button-primary" onClick={submit} disabled={saving || !plugins.length}>
          {saving ? <Loader2 size={16} className="ds-spin" /> : <Plug size={16} />} 保存
        </button>
        {saved ? <span className="badge badge-success"><Check size={13} /> 已保存</span> : null}
      </div>

      <Modal
        open={showImport}
        title="导入插件"
        subtitle="JSX 源码将在隔离沙箱中运行；导入后默认关闭。"
        width={640}
        onClose={() => setShowImport(false)}
        footer={
          <>
            <button className="button" onClick={() => setShowImport(false)}>取消</button>
            <button className="button button-primary" onClick={doImport} disabled={importing}>
              {importing ? <Loader2 size={15} className="ds-spin" /> : <Plus size={15} />} 导入（默认关闭）
            </button>
          </>
        }
      >
        {importErr ? <div className="panel badge-danger" style={{ borderRadius: 10, marginBottom: 10 }}>{importErr}</div> : null}
        <div style={{ display: "grid", gap: 12 }}>
              <div style={{ display: "flex", gap: 10 }}>
                <div className="field" style={{ flex: 1 }}>
                  <label>Key（小写、连字符）</label>
                  <input value={draft.key} placeholder="my-plugin" onChange={(e) => setDraft({ ...draft, key: e.target.value })} />
                </div>
                <div className="field" style={{ flex: 1 }}>
                  <label>名称</label>
                  <input value={draft.name} placeholder="我的插件" onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
                </div>
              </div>
              <div className="field">
                <label>说明（可选）</label>
                <input value={draft.description} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
              </div>
              <div style={{ display: "flex", gap: 10 }}>
                <div className="field" style={{ flex: 1 }}>
                  <label>类型</label>
                  <select value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value as Draft["kind"] })}>
                    <option value="component">组件（&lt;Tag/&gt;）</option>
                    <option value="fence">围栏（```lang）</option>
                  </select>
                </div>
                <div className="field" style={{ flex: 1 }}>
                  <label>分类</label>
                  <select value={draft.category} onChange={(e) => setDraft({ ...draft, category: e.target.value })}>
                    {Object.entries(CATEGORY_LABELS).map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                  </select>
                </div>
                <div className="field" style={{ flex: 1 }}>
                  {draft.kind === "component" ? (
                    <>
                      <label>标签名（大写开头）</label>
                      <input value={draft.tag} placeholder="MyTag" onChange={(e) => setDraft({ ...draft, tag: e.target.value })} />
                    </>
                  ) : (
                    <>
                      <label>围栏语言（小写）</label>
                      <input value={draft.lang} placeholder="mylang" onChange={(e) => setDraft({ ...draft, lang: e.target.value })} />
                    </>
                  )}
                </div>
              </div>
              <div className="field">
                <label>JSX 源码（需定义 <code>function Plugin(props)</code>）</label>
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
