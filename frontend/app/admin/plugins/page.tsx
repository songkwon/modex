"use client";

import { useEffect, useMemo, useState } from "react";
import { Check, Loader2, Plug } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { getPlugins, savePlugins, type PluginState, type PluginConfig } from "@/lib/api";

const CATEGORY_LABELS: Record<string, string> = {
  diagram: "图表",
  math: "数学",
  content: "内容增强",
  api: "API"
};

export default function AdminPluginsPage() {
  const [plugins, setPlugins] = useState<PluginState[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    getPlugins()
      .then((r) => setPlugins(r.plugins || []))
      .catch((e) => setError(String(e)));
  }, []);

  const groups = useMemo(() => {
    const order = ["diagram", "math", "content", "api"];
    const by: Record<string, PluginState[]> = {};
    for (const p of plugins) (by[p.category] ||= []).push(p);
    return order.filter((c) => by[c]?.length).map((c) => [c, by[c]] as const);
  }, [plugins]);

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

  return (
    <AdminShell
      title="插件管理"
      kicker="Plugins"
      description="开启或关闭文档渲染引擎的内置能力，并配置参数。改动保存后立即对全站文档生效。"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      {groups.map(([category, items]) => (
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

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button className="button button-primary" onClick={submit} disabled={saving || !plugins.length}>
          {saving ? <Loader2 size={16} className="ds-spin" /> : <Plug size={16} />} 保存
        </button>
        {saved ? <span className="badge badge-success"><Check size={13} /> 已保存</span> : null}
      </div>
    </AdminShell>
  );
}
