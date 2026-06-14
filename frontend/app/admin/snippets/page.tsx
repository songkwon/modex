"use client";

import { useEffect, useState } from "react";
import { Check, Loader2, Plus, Puzzle, Trash2 } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { getSnippets, saveSnippets, type Snippet } from "@/lib/api";

type VarRow = { key: string; value: string };

export default function AdminSnippetsPage() {
  const [snippets, setSnippets] = useState<Snippet[]>([]);
  const [vars, setVars] = useState<VarRow[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    getSnippets()
      .then((d) => {
        setSnippets(d.snippets || []);
        setVars(Object.entries(d.variables || {}).map(([key, value]) => ({ key, value })));
      })
      .catch((e) => setError(String(e)));
  }, []);

  function patchSnippet(i: number, patch: Partial<Snippet>) {
    setSnippets((s) => s.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  }
  function patchVar(i: number, patch: Partial<VarRow>) {
    setVars((v) => v.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));
  }

  async function submit() {
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      const variables: Record<string, string> = {};
      for (const v of vars) if (v.key.trim()) variables[v.key.trim()] = v.value;
      const d = await saveSnippets({ snippets: snippets.filter((s) => s.key.trim()), variables });
      setSnippets(d.snippets || []);
      setVars(Object.entries(d.variables || {}).map(([key, value]) => ({ key, value })));
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
      title="复用片段"
      kicker="Snippets"
      description="定义可在任意文档中复用的片段（<Snippet name=&quot;key&quot;/>）与全局变量（{{key}}）。保存后立即对全站文档生效。"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <section className="card" style={{ display: "grid", gap: 14 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h2 className="page-kicker" style={{ margin: 0 }}>片段</h2>
          <button className="button" onClick={() => setSnippets((s) => [...s, { key: "", name: "", content: "" }])}>
            <Plus size={15} /> 新增片段
          </button>
        </div>
        {snippets.length === 0 ? <p className="muted" style={{ fontSize: 13 }}>暂无片段。</p> : null}
        {snippets.map((s, i) => (
          <div key={i} className="snippet-row">
            <div style={{ display: "flex", gap: 8 }}>
              <input style={{ flex: "0 0 200px" }} value={s.key} placeholder="key（引用名）" onChange={(e) => patchSnippet(i, { key: e.target.value })} />
              <input style={{ flex: 1 }} value={s.name} placeholder="显示名称（可选）" onChange={(e) => patchSnippet(i, { name: e.target.value })} />
              <button className="button" aria-label="删除" onClick={() => setSnippets((arr) => arr.filter((_, idx) => idx !== i))}>
                <Trash2 size={15} />
              </button>
            </div>
            <textarea
              style={{ minHeight: 96, padding: "10px 12px", lineHeight: 1.6, fontFamily: "ui-monospace, monospace", fontSize: 13 }}
              value={s.content}
              placeholder="Markdown / MDX 内容，可包含 {{变量}} 或嵌套 <Snippet/>"
              onChange={(e) => patchSnippet(i, { content: e.target.value })}
            />
          </div>
        ))}
      </section>

      <section className="card" style={{ display: "grid", gap: 14 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h2 className="page-kicker" style={{ margin: 0 }}>全局变量</h2>
          <button className="button" onClick={() => setVars((v) => [...v, { key: "", value: "" }])}>
            <Plus size={15} /> 新增变量
          </button>
        </div>
        {vars.length === 0 ? <p className="muted" style={{ fontSize: 13 }}>暂无变量。</p> : null}
        {vars.map((v, i) => (
          <div key={i} style={{ display: "flex", gap: 8 }}>
            <input style={{ flex: "0 0 200px" }} value={v.key} placeholder="key" onChange={(e) => patchVar(i, { key: e.target.value })} />
            <input style={{ flex: 1 }} value={v.value} placeholder="值" onChange={(e) => patchVar(i, { value: e.target.value })} />
            <button className="button" aria-label="删除" onClick={() => setVars((arr) => arr.filter((_, idx) => idx !== i))}>
              <Trash2 size={15} />
            </button>
          </div>
        ))}
      </section>

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button className="button button-primary" onClick={submit} disabled={saving}>
          {saving ? <Loader2 size={16} className="ds-spin" /> : <Puzzle size={16} />} 保存
        </button>
        {saved ? <span className="badge badge-success"><Check size={13} /> 已保存</span> : null}
      </div>
    </AdminShell>
  );
}
