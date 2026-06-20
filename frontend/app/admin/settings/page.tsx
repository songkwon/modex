"use client";

import { useEffect, useState } from "react";
import { Check, KeyRound, ListChecks, Loader2, RotateCcw, Sparkles } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Combobox } from "@/components/ui/combobox";
import { fetchModels, getSettings, runRecallTest, saveSettings, type AISettings, type RecallTestResult } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

// Supported chat API formats. "openai-chat" covers OpenAI and every
// OpenAI-compatible vendor (DeepSeek, Qwen, GLM, Moonshot, Ollama, vLLM, …);
// the others are the native protocols of the major providers.
const DEFAULT_AI: AISettings = { ask_protocol: "openai-chat", ask_base_url: "", ask_model: "", ask_api_key: "", ask_system_prompt: "" };

export default function AdminSettingsPage() {
  const { t } = useI18n();
  const PROTOCOLS: { value: string; label: string }[] = [
    { value: "openai-chat", label: "OpenAI Chat Completions" },
    { value: "anthropic", label: t("legacy.d4030608fc9d") },
    { value: "gemini", label: t("legacy.5436dfa7256d") },
    { value: "openai-responses", label: "OpenAI Responses API" },
  ];
  // Provider presets only fill the API format + Base URL — the model is always
  // fetched from the endpoint, never hardcoded. Both fields stay editable.
  const PRESETS: { label: string; base: string; protocol: string }[] = [
    { label: "OpenAI", base: "https://api.openai.com/v1", protocol: "openai-chat" },
    { label: "Anthropic Claude", base: "https://api.anthropic.com", protocol: "anthropic" },
    { label: "Google Gemini", base: "https://generativelanguage.googleapis.com", protocol: "gemini" },
    { label: "DeepSeek", base: "https://api.deepseek.com/v1", protocol: "openai-chat" },
    { label: t("legacy.b2e4f4b07a03"), base: "https://dashscope.aliyuncs.com/compatible-mode/v1", protocol: "openai-chat" },
    { label: t("legacy.3777d3667de2"), base: "https://open.bigmodel.cn/api/paas/v4", protocol: "openai-chat" },
    { label: "Kimi (Moonshot)", base: "https://api.moonshot.cn/v1", protocol: "openai-chat" },
    { label: "SiliconFlow", base: "https://api.siliconflow.cn/v1", protocol: "openai-chat" },
    { label: t("legacy.5680cae9fb6e"), base: "http://localhost:11434/v1", protocol: "openai-chat" },
    { label: t("legacy.e0be05c80ca4"), base: "http://localhost:8000/v1", protocol: "openai-chat" },
  ];
  const CHUNK_STRATEGIES = [
    { value: "markdown", label: t("legacy.8e5f107f6b1d") },
    { value: "heading", label: t("legacy.51f3e00a19a8") },
    { value: "fixed", label: t("legacy.c55a5a68d5f9") },
    { value: "semantic", label: t("legacy.594f1e850c4c") },
  ];
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
    if (!ai.ask_base_url) { setError(t("legacy.9250789d08a7")); return; }
    setLoadingModels(true);
    setError("");
    try {
      const r = await fetchModels(ai.ask_base_url, ai.ask_api_key, protocol);
      setModels(r.models || []);
      if (!r.models?.length) setError(t("legacy.cbb74d64d1e7"));
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
      title={t("legacy.082a738ae620")}
      kicker="AI Model"
      description={t("legacy.ae0a8bc19bf9")}
      contentClassName="admin-settings-content"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <section className="card" style={{ display: "grid", gap: 18 }}>
        <div className="field">
          <label>{t("legacy.774abe01308a")}</label>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
            {PRESETS.map((p) => (
              <button key={p.label} className="shelf-tab" onClick={() => { setModels([]); setAI({ ...ai, ask_base_url: p.base, ask_protocol: p.protocol, ask_model: "" }); }}>
                {p.label}
              </button>
            ))}
          </div>
          <span className="field-hint">{t("legacy.1c03666d5421")}</span>
        </div>

        <div className="field">
          <label>{t("legacy.6e8df53870d9")}</label>
          <Combobox
            options={PROTOCOLS}
            value={[protocol]}
            onChange={(v) => { setModels([]); setAI({ ...ai, ask_protocol: v[0] || "openai-chat", ask_model: "" }); }}
            multiple={false}
            placeholder={t("legacy.802c7a7a616c")}
          />
          <span className="field-hint">{t("legacy.0ac0b795b3d9")} <code className="code-chip">OpenAI Chat Completions</code>。</span>
        </div>

        <div className="field">
          <label>API Base URL</label>
          <input value={ai.ask_base_url || ""} placeholder="https://api.openai.com/v1" onChange={(e) => setAI({ ...ai, ask_base_url: e.target.value })} />
          <span className="field-hint">{t("legacy.bc7a7d38b333")} <code className="code-chip">/chat/completions</code>、<code className="code-chip">/v1/messages</code>）。</span>
        </div>

        <div className="field">
          <label>API Key</label>
          <div style={{ position: "relative" }}>
            <KeyRound size={15} style={{ position: "absolute", left: 12, top: 13, color: "hsl(var(--muted))" }} />
            <input
              type="password"
              style={{ paddingLeft: 34 }}
              value={ai.ask_api_key || ""}
              placeholder={keySet ? t("legacy.1b34ab64cb4c") : "sk-…"}
              onChange={(e) => setAI({ ...ai, ask_api_key: e.target.value })}
            />
          </div>
          {keySet ? <span className="field-hint" style={{ color: "hsl(var(--accent-strong))" }}>{t("legacy.fcd5a8a4f116")}</span> : null}
        </div>

        <div className="field">
          <label>{t("legacy.c98e118e0a43")}</label>
          <div style={{ display: "flex", gap: 8, alignItems: "flex-start" }}>
            <div style={{ flex: 1 }}>
              <Combobox
                options={models.map((m) => ({ value: m, label: m }))}
                value={ai.ask_model ? [ai.ask_model] : []}
                onChange={(v) => setAI({ ...ai, ask_model: v[0] || "" })}
                multiple={false}
                allowCreate
                placeholder={models.length ? t("legacy.13a1ec97c00b") : t("legacy.81b332916aca")}
              />
            </div>
            <button className="button" onClick={loadModels} disabled={loadingModels} style={{ flex: "none", height: 42 }}>
              {loadingModels ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} {t("legacy.370174b64fed")}
            </button>
          </div>
          <span className="field-hint">{t("legacy.10e1f3c4399f")}</span>
        </div>

        <div style={{ display: "flex", gap: 16 }}>
          <div className="field" style={{ flex: 1 }}>
            <label>{t("legacy.1e60c0ddf4bf")}</label>
            <input
              type="number"
              min={1}
              value={ai.ask_max_tokens ?? ""}
              placeholder={t("legacy.11cb7ad4c567")}
              onChange={(e) => setAI({ ...ai, ask_max_tokens: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
            />
            <span className="field-hint">{t("legacy.2793f5c020db")}</span>
          </div>
          <div className="field" style={{ flex: 1 }}>
            <label>{t("legacy.a9ec1c08515f")}</label>
            <input
              type="number"
              min={0}
              max={2}
              step={0.1}
              value={ai.ask_temperature ?? ""}
              placeholder={t("legacy.390e62264453")}
              onChange={(e) => setAI({ ...ai, ask_temperature: e.target.value === "" ? undefined : parseFloat(e.target.value) })}
            />
            <span className="field-hint">{t("legacy.a2d1c9e11272")}</span>
          </div>
        </div>

        <div className="field">
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 8 }}>
            <label style={{ margin: 0 }}>{t("legacy.0894b78b866e")}</label>
            <button
              type="button"
              className="shelf-tab"
              onClick={() => setAI({ ...ai, ask_system_prompt: promptDefault })}
              disabled={!promptDefault || ai.ask_system_prompt === promptDefault}
              title={t("legacy.0e1291fd3454")}
              style={{ display: "inline-flex", alignItems: "center", gap: 6 }}
            >
              <RotateCcw size={13} /> {t("legacy.08a815d9ce91")}
            </button>
          </div>
          <textarea
            style={{ minHeight: 120, padding: "10px 12px", lineHeight: 1.6 }}
            value={ai.ask_system_prompt || ""}
            placeholder={promptDefault || t("legacy.01a66986ddcb")}
            onChange={(e) => setAI({ ...ai, ask_system_prompt: e.target.value })}
          />
          <span className="field-hint">{t("legacy.3c86d2b2decd")}</span>
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.11497d72d8d8")}</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>Embedding Base URL</label>
              <input
                value={ai.embedding_base_url || ""}
                placeholder="https://api.example.com/v1"
                onChange={(e) => setAI({ ...ai, embedding_base_url: e.target.value })}
              />
              <span className="field-hint">{t("legacy.2dfb5037af44")}</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.6ac9d692b3a5")}</label>
              <input
                value={ai.embedding_model || ""}
                placeholder="text-embedding-3-large / bge-m3"
                onChange={(e) => setAI({ ...ai, embedding_model: e.target.value })}
              />
              <span className="field-hint">{t("legacy.bda9b59f36e9")}</span>
            </div>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>Embedding API Key</label>
              <input
                type="password"
                value={ai.embedding_api_key || ""}
                placeholder={embeddingKeySet ? t("legacy.1b34ab64cb4c") : "sk-…"}
                onChange={(e) => setAI({ ...ai, embedding_api_key: e.target.value })}
              />
              <span className="field-hint">{t("legacy.c0988d287a6f")}</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.9a9ec48b54a6")}</label>
              <input type="number" value={1024} readOnly disabled />
              <span className="field-hint">{t("legacy.6548e49aedc6")}</span>
            </div>
          </div>
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.94f35bc53fe7")}</h2>
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
              <label>{t("legacy.384705c5fe9d")}</label>
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
                placeholder={rerankKeySet ? t("legacy.1b34ab64cb4c") : "sk-…"}
                onChange={(e) => setAI({ ...ai, rerank_api_key: e.target.value })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.790772caedba")}</label>
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
              <label>{t("legacy.c3b28603f37a")}</label>
              <input
                value={ai.recall_test_query || ""}
                placeholder={t("legacy.bc0a4c4b3e24")}
                onChange={(e) => setAI({ ...ai, recall_test_query: e.target.value })}
              />
              <span className="field-hint">{t("legacy.91a8a39cced9")}</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.99c19036e70a")}</label>
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
            <label>{t("legacy.3555c8aa3d59")}</label>
            <textarea
              style={{ minHeight: 78, padding: "10px 12px", lineHeight: 1.6 }}
              value={ai.recall_test_doc_ids || ""}
              placeholder="DemoModule:latest:guide"
              onChange={(e) => setAI({ ...ai, recall_test_doc_ids: e.target.value })}
            />
            <span className="field-hint">{t("legacy.1daaf401584f")}</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <button className="button" onClick={handleRecallTest} disabled={recallLoading}>
              {recallLoading ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} {t("legacy.ffe7c281276a")}
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
                <span className="tag">{t("legacy.393df9bb13ea")} {recallResult.actual_doc_ids.filter((id) => recallResult.expected_doc_ids.includes(id)).length}/{recallResult.expected_doc_ids.length}</span>
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
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.fe1084a76375")}</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.9c8eb75c7e58")}</label>
              <Combobox
                options={CHUNK_STRATEGIES}
                value={ai.chunk_strategy ? [ai.chunk_strategy] : []}
                onChange={(v) => setAI({ ...ai, chunk_strategy: v[0] || "" })}
                multiple={false}
                placeholder={t("legacy.b1b25cf38535")}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.bffabb2cdf5d")}</label>
              <input
                type="number"
                min={1}
                value={ai.chunk_size ?? ""}
                placeholder="800"
                onChange={(e) => setAI({ ...ai, chunk_size: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("legacy.be8a228b6ee0")}</label>
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
            {saving ? <Loader2 size={16} className="ds-spin" /> : ''} {t("legacy.c8550237ba70")}
          </button>
          {saved ? <span className="badge badge-success"><Check size={13} /> {t("legacy.1bd91a7d0c53")}</span> : null}
        </div>
      </section>

      <p className="muted" style={{ fontSize: 12 }}>
        {t("legacy.455bb89e393f")}
      </p>
    </AdminShell>
  );
}
