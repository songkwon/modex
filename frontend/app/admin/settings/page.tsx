"use client";

import { useEffect, useState } from "react";
import { Check, KeyRound, ListChecks, Loader2, RotateCcw, Sparkles } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Combobox } from "@/components/ui/combobox";
import { fetchModels, getSettings, runRecallTest, saveSettings, type AISettings, type RecallTestResult } from "@/lib/api";

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
const CHUNK_STRATEGIES = [
  { value: "markdown", label: "Markdown 标题优先" },
  { value: "heading", label: "标题层级" },
  { value: "fixed", label: "固定长度" },
  { value: "semantic", label: "语义分段" },
];

export default function AdminSettingsPage() {
  const [ai, setAI] = useState<AISettings>(DEFAULT_AI);
  const [keySet, setKeySet] = useState(false);
  const [embeddingKeySet, setEmbeddingKeySet] = useState(false);
  const [rerankKeySet, setRerankKeySet] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const [models, setModels] = useState<string[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [promptDefault, setPromptDefault] = useState("");
  const [recallLoading, setRecallLoading] = useState(false);
  const [recallResult, setRecallResult] = useState<RecallTestResult | null>(null);

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
        setPromptDefault(s.ask_system_prompt_default || "");
        setAI({
          ...DEFAULT_AI,
          ...s.ai,
          ask_api_key: "",
          // Pre-fill the built-in prompt so admins edit on top of it instead of a
          // blank box. An empty stored value means "use the default".
          ask_system_prompt: s.ai?.ask_system_prompt || s.ask_system_prompt_default || "",
        });
        setKeySet(s.ask_api_key_set);
        setEmbeddingKeySet(Boolean(s.embedding_api_key_set));
        setRerankKeySet(Boolean(s.rerank_api_key_set));
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
      if (!payload.embedding_api_key) delete payload.embedding_api_key;
      if (!payload.rerank_api_key) delete payload.rerank_api_key;
      const s = await saveSettings(payload);
      setAI({ ...DEFAULT_AI, ...s.ai, ask_api_key: "" });
      setKeySet(s.ask_api_key_set);
      setEmbeddingKeySet(Boolean(s.embedding_api_key_set));
      setRerankKeySet(Boolean(s.rerank_api_key_set));
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  }

  async function handleRecallTest() {
    setRecallLoading(true);
    setError("");
    setRecallResult(null);
    try {
      const payload: AISettings = { ...ai };
      if (!payload.ask_api_key) delete payload.ask_api_key;
      if (!payload.embedding_api_key) delete payload.embedding_api_key;
      if (!payload.rerank_api_key) delete payload.rerank_api_key;
      await saveSettings(payload);
      const r = await runRecallTest();
      setRecallResult(r);
    } catch (e) {
      setError(String(e));
    } finally {
      setRecallLoading(false);
    }
  }

  return (
    <AdminShell
      title="模型设置"
      kicker="AI Model"
      description="配置对话、嵌入、重排序模型，以及 RAG 召回测试和文档分段策略。"
      contentClassName="admin-settings-content"
    >
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
          <span className="field-hint">决定请求的端点与报文格式。大多数兼容服务用 <code className="code-chip">OpenAI Chat Completions</code>。</span>
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

        <div style={{ display: "flex", gap: 16 }}>
          <div className="field" style={{ flex: 1 }}>
            <label>最大回复 Tokens（可选）</label>
            <input
              type="number"
              min={1}
              value={ai.ask_max_tokens ?? ""}
              placeholder="默认 4096（Anthropic 必填，其它留空不限）"
              onChange={(e) => setAI({ ...ai, ask_max_tokens: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
            />
            <span className="field-hint">控制单次回答长度上限，避免长回答被截断。</span>
          </div>
          <div className="field" style={{ flex: 1 }}>
            <label>采样温度 Temperature（可选）</label>
            <input
              type="number"
              min={0}
              max={2}
              step={0.1}
              value={ai.ask_temperature ?? ""}
              placeholder="默认 0.2"
              onChange={(e) => setAI({ ...ai, ask_temperature: e.target.value === "" ? undefined : parseFloat(e.target.value) })}
            />
            <span className="field-hint">越低越确定、越贴合文档；越高越发散。</span>
          </div>
        </div>

        <div className="field">
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 8 }}>
            <label style={{ margin: 0 }}>系统提示词</label>
            <button
              type="button"
              className="shelf-tab"
              onClick={() => setAI({ ...ai, ask_system_prompt: promptDefault })}
              disabled={!promptDefault || ai.ask_system_prompt === promptDefault}
              title="恢复为内置默认提示词"
              style={{ display: "inline-flex", alignItems: "center", gap: 6 }}
            >
              <RotateCcw size={13} /> 重置为默认
            </button>
          </div>
          <textarea
            style={{ minHeight: 120, padding: "10px 12px", lineHeight: 1.6 }}
            value={ai.ask_system_prompt || ""}
            placeholder={promptDefault || "留空使用内置中文 RAG 提示词"}
            onChange={(e) => setAI({ ...ai, ask_system_prompt: e.target.value })}
          />
          <span className="field-hint">已预填内置默认提示词，可在此基础上修改；点「重置为默认」可还原。留空保存则回退到内置版本。</span>
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>嵌入模型</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>Embedding Base URL</label>
              <input
                value={ai.embedding_base_url || ""}
                placeholder="https://api.example.com/v1"
                onChange={(e) => setAI({ ...ai, embedding_base_url: e.target.value })}
              />
              <span className="field-hint">用于语义检索和混合检索的向量生成接口。</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>Embedding 模型</label>
              <input
                value={ai.embedding_model || ""}
                placeholder="text-embedding-3-large / bge-m3"
                onChange={(e) => setAI({ ...ai, embedding_model: e.target.value })}
              />
              <span className="field-hint">模型名按供应商接口填写。</span>
            </div>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>Embedding API Key</label>
              <input
                type="password"
                value={ai.embedding_api_key || ""}
                placeholder={embeddingKeySet ? "已配置（留空则保持不变）" : "sk-…"}
                onChange={(e) => setAI({ ...ai, embedding_api_key: e.target.value })}
              />
              <span className="field-hint">嵌入接口的访问密钥；留空保持原有配置不变。</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>向量维度（固定）</label>
              <input type="number" value={1024} readOnly disabled />
              <span className="field-hint">系统固定为 1024 维，请选用输出 1024 维的模型（如 bge-m3）。</span>
            </div>
          </div>
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>重排序与召回测试</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>Rerank Base URL</label>
              <input
                value={ai.rerank_base_url || ""}
                placeholder="https://api.example.com/v1"
                onChange={(e) => setAI({ ...ai, rerank_base_url: e.target.value })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>Rerank 模型</label>
              <input
                value={ai.rerank_model || ""}
                placeholder="bge-reranker-v2-m3"
                onChange={(e) => setAI({ ...ai, rerank_model: e.target.value })}
              />
            </div>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>Rerank API Key</label>
              <input
                type="password"
                value={ai.rerank_api_key || ""}
                placeholder={rerankKeySet ? "已配置（留空则保持不变）" : "sk-…"}
                onChange={(e) => setAI({ ...ai, rerank_api_key: e.target.value })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>重排 TopK</label>
              <input
                type="number"
                min={1}
                value={ai.rerank_top_k ?? ""}
                placeholder="20"
                onChange={(e) => setAI({ ...ai, rerank_top_k: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>召回测试 Query</label>
              <input
                value={ai.recall_test_query || ""}
                placeholder="构建缓存怎么清理"
                onChange={(e) => setAI({ ...ai, recall_test_query: e.target.value })}
              />
              <span className="field-hint">保存一条基准问题，后续用于对比不同分段、向量和重排配置。</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>召回测试 TopK</label>
              <input
                type="number"
                min={1}
                value={ai.recall_test_top_k ?? ""}
                placeholder="10"
                onChange={(e) => setAI({ ...ai, recall_test_top_k: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
          </div>
          <div className="field">
            <label>期望命中文档 ID</label>
            <textarea
              style={{ minHeight: 78, padding: "10px 12px", lineHeight: 1.6 }}
              value={ai.recall_test_doc_ids || ""}
              placeholder="DemoModule:latest:guide"
              onChange={(e) => setAI({ ...ai, recall_test_doc_ids: e.target.value })}
            />
            <span className="field-hint">一行一个 doc_id，后续召回测试按命中率、MRR、nDCG 展示效果。</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <button className="button" onClick={handleRecallTest} disabled={recallLoading}>
              {recallLoading ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} 运行召回测试
            </button>
            {recallResult ? (
              <span className="badge badge-success">
                Recall@{recallResult.top_k}: {(recallResult.recall_at_k * 100).toFixed(1)}%
              </span>
            ) : null}
          </div>
          {recallResult ? (
            <div className="panel" style={{ display: "grid", gap: 10, borderRadius: 12, padding: 12 }}>
              <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                <span className="tag">MRR {recallResult.mrr.toFixed(3)}</span>
                <span className="tag">nDCG {recallResult.ndcg.toFixed(3)}</span>
                <span className="tag">命中 {recallResult.actual_doc_ids.filter((id) => recallResult.expected_doc_ids.includes(id)).length}/{recallResult.expected_doc_ids.length}</span>
              </div>
              <ol style={{ margin: 0, paddingLeft: 20, display: "grid", gap: 6 }}>
                {(recallResult.results || []).slice(0, recallResult.top_k).map((r) => (
                  <li key={r.doc_id} style={{ fontSize: 13 }}>
                    <code className="code-chip">{r.doc_id}</code>
                    <span style={{ marginLeft: 8 }}>{r.title}</span>
                    <span className="muted" style={{ marginLeft: 8 }}>score {r.score.toFixed(3)}</span>
                  </li>
                ))}
              </ol>
              {recallResult.note ? <span className="field-hint">{recallResult.note}</span> : null}
            </div>
          ) : null}
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>分段策略</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>策略</label>
              <Combobox
                options={CHUNK_STRATEGIES}
                value={ai.chunk_strategy ? [ai.chunk_strategy] : []}
                onChange={(v) => setAI({ ...ai, chunk_strategy: v[0] || "" })}
                multiple={false}
                placeholder="选择分段策略…"
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>分段长度</label>
              <input
                type="number"
                min={1}
                value={ai.chunk_size ?? ""}
                placeholder="800"
                onChange={(e) => setAI({ ...ai, chunk_size: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>重叠长度</label>
              <input
                type="number"
                min={0}
                value={ai.chunk_overlap ?? ""}
                placeholder="120"
                onChange={(e) => setAI({ ...ai, chunk_overlap: e.target.value === "" ? undefined : Math.max(0, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
          </div>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <button className="button button-primary" onClick={submit} disabled={saving}>
            {saving ? <Loader2 size={16} className="ds-spin" /> : ''} 保存设置
          </button>
          {saved ? <span className="badge badge-success"><Check size={13} /> 已保存</span> : null}
        </div>
      </section>

      <p className="muted" style={{ fontSize: 12 }}>
        提示：对话模型保存后立即用于 AI 问答；嵌入、重排序、分段和召回测试配置会随平台设置持久化，索引链路接入后直接复用。
      </p>
    </AdminShell>
  );
}
