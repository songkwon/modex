"use client";

import { useEffect, useState } from "react";
import { Check, KeyRound, ListChecks, Loader2, Sparkles } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Combobox } from "@/components/ui/combobox";
import { fetchModels, getSettings, saveSettings, type AISettings } from "@/lib/api";

// Supported chat API formats. "openai-chat" covers OpenAI and every
// OpenAI-compatible vendor (DeepSeek, Qwen, GLM, Moonshot, Ollama, vLLM, …);
// the others are the native protocols of the major providers.
const PROTOCOLS: { value: string; label: string }[] = [
  { value: "openai-chat", label: "OpenAI Chat Completions" },
  { value: "anthropic", label: "Anthropic Messages（原生）" },
  { value: "gemini", label: "Gemini generateContent（原生）" },
  { value: "openai-responses", label: "OpenAI Responses API" },
];

// Provider presets only fill the API format + Base URL — the model is always
// fetched from the endpoint, never hardcoded. Both fields stay editable.
const PRESETS: { label: string; base: string; protocol: string }[] = [
  { label: "OpenAI", base: "https://api.openai.com/v1", protocol: "openai-chat" },
  { label: "Anthropic Claude", base: "https://api.anthropic.com", protocol: "anthropic" },
  { label: "Google Gemini", base: "https://generativelanguage.googleapis.com", protocol: "gemini" },
  { label: "DeepSeek", base: "https://api.deepseek.com/v1", protocol: "openai-chat" },
  { label: "通义千问 Qwen", base: "https://dashscope.aliyuncs.com/compatible-mode/v1", protocol: "openai-chat" },
  { label: "智谱 GLM", base: "https://open.bigmodel.cn/api/paas/v4", protocol: "openai-chat" },
  { label: "Kimi (Moonshot)", base: "https://api.moonshot.cn/v1", protocol: "openai-chat" },
  { label: "SiliconFlow", base: "https://api.siliconflow.cn/v1", protocol: "openai-chat" },
  { label: "Ollama（本地）", base: "http://localhost:11434/v1", protocol: "openai-chat" },
  { label: "vLLM（自托管）", base: "http://localhost:8000/v1", protocol: "openai-chat" },
];

const DEFAULT_AI: AISettings = { ask_protocol: "openai-chat", ask_base_url: "", ask_model: "", ask_api_key: "", ask_system_prompt: "" };

export default function AdminSettingsPage() {
  const [ai, setAI] = useState<AISettings>(DEFAULT_AI);
  const [keySet, setKeySet] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const [models, setModels] = useState<string[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);

  const protocol = ai.ask_protocol || "openai-chat";

  async function loadModels() {
    if (!ai.ask_base_url) { setError("请先填写 API Base URL"); return; }
    setLoadingModels(true);
    setError("");
    try {
      const r = await fetchModels(ai.ask_base_url, ai.ask_api_key, protocol);
      setModels(r.models || []);
      if (!r.models?.length) setError("该端点未返回模型列表，请确认地址、密钥和 API 格式是否匹配。");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingModels(false);
    }
  }

  useEffect(() => {
    getSettings()
      .then((s) => {
        setAI({ ...DEFAULT_AI, ...s.ai, ask_api_key: "" });
        setKeySet(s.ask_api_key_set);
      })
      .catch((e) => setError(String(e)));
  }, []);

  async function submit() {
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      const payload: AISettings = { ...ai };
      if (!payload.ask_api_key) delete payload.ask_api_key; // keep existing key
      const s = await saveSettings(payload);
      setAI({ ...DEFAULT_AI, ...s.ai, ask_api_key: "" });
      setKeySet(s.ask_api_key_set);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <AdminShell title="模型设置" kicker="AI Model" description="对接对话模型用于全站 AI 问答（RAG）。支持 OpenAI、Anthropic、Gemini 等主流 API 格式与任意兼容端点；未配置时回退到基于检索的摘要式回答。">
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <section className="card" style={{ display: "grid", gap: 18 }}>
        <div className="field">
          <label>快速预设</label>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
            {PRESETS.map((p) => (
              <button key={p.label} className="shelf-tab" onClick={() => { setModels([]); setAI({ ...ai, ask_base_url: p.base, ask_protocol: p.protocol, ask_model: "" }); }}>
                {p.label}
              </button>
            ))}
          </div>
          <span className="field-hint">选择预设只填入 API 格式与 Base URL，模型请点「获取模型列表」从接口拉取。</span>
        </div>

        <div className="field">
          <label>API 格式</label>
          <Combobox
            options={PROTOCOLS}
            value={[protocol]}
            onChange={(v) => { setModels([]); setAI({ ...ai, ask_protocol: v[0] || "openai-chat", ask_model: "" }); }}
            multiple={false}
            placeholder="选择 API 格式…"
          />
          <span className="field-hint">决定请求的端点与报文格式。大多数国产/开源服务用 <code className="code-chip">OpenAI Chat Completions</code>。</span>
        </div>

        <div className="field">
          <label>API Base URL</label>
          <input value={ai.ask_base_url || ""} placeholder="https://api.openai.com/v1" onChange={(e) => setAI({ ...ai, ask_base_url: e.target.value })} />
          <span className="field-hint">服务的根地址；不同 API 格式会自动追加各自的路径（如 <code className="code-chip">/chat/completions</code>、<code className="code-chip">/v1/messages</code>）。</span>
        </div>

        <div className="field">
          <label>API Key</label>
          <div style={{ position: "relative" }}>
            <KeyRound size={15} style={{ position: "absolute", left: 12, top: 13, color: "hsl(var(--muted))" }} />
            <input
              type="password"
              style={{ paddingLeft: 34 }}
              value={ai.ask_api_key || ""}
              placeholder={keySet ? "已配置（留空则保持不变）" : "sk-…"}
              onChange={(e) => setAI({ ...ai, ask_api_key: e.target.value })}
            />
          </div>
          {keySet ? <span className="field-hint" style={{ color: "hsl(var(--accent-strong))" }}>已保存密钥；如需更换请输入新值。</span> : null}
        </div>

        <div className="field">
          <label>模型</label>
          <div style={{ display: "flex", gap: 8, alignItems: "flex-start" }}>
            <div style={{ flex: 1 }}>
              <Combobox
                options={models.map((m) => ({ value: m, label: m }))}
                value={ai.ask_model ? [ai.ask_model] : []}
                onChange={(v) => setAI({ ...ai, ask_model: v[0] || "" })}
                multiple={false}
                allowCreate
                placeholder={models.length ? "选择模型…" : "点右侧「获取模型列表」从接口拉取"}
              />
            </div>
            <button className="button" onClick={loadModels} disabled={loadingModels} style={{ flex: "none", height: 42 }}>
              {loadingModels ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} 获取模型列表
            </button>
          </div>
          <span className="field-hint">模型从接口实时获取；填好 Base URL 与 Key 后点击拉取。</span>
        </div>

        <div className="field">
          <label>系统提示词（可选）</label>
          <textarea
            style={{ minHeight: 84, padding: "10px 12px", lineHeight: 1.6 }}
            value={ai.ask_system_prompt || ""}
            placeholder="留空使用内置中文 RAG 提示词"
            onChange={(e) => setAI({ ...ai, ask_system_prompt: e.target.value })}
          />
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <button className="button button-primary" onClick={submit} disabled={saving}>
            {saving ? <Loader2 size={16} className="ds-spin" /> : <Sparkles size={16} />} 保存设置
          </button>
          {saved ? <span className="badge badge-success"><Check size={13} /> 已保存</span> : null}
        </div>
      </section>

      <p className="muted" style={{ fontSize: 12 }}>
        提示：对话模型在此页配置后立即生效，无需重启。向量 / 重排序模型配置将在后续版本加入。
      </p>
    </AdminShell>
  );
}
