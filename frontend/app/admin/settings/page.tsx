"use client";

import { useEffect, useState } from "react";
import { Check, KeyRound, ListChecks, Loader2, RotateCcw, Sparkles } from "lucide-react";
import { AdminShell } from "@/components/admin-shell";
import { Combobox } from "@/components/ui/combobox";
import { fetchModels, getSettings, runRecallTest, saveSettings, testModelConnection, type AISettings, type ModelConnectionTestResult, type RecallTestResult } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

// Supported chat API formats. "openai-chat" covers OpenAI and every
// OpenAI-compatible vendor (DeepSeek, Qwen, GLM, Moonshot, Ollama, vLLM, …);
// the others are the native protocols of the major providers.
const DEFAULT_AI: AISettings = { ask_protocol: "openai-chat", ask_base_url: "", ask_model: "", ask_api_key: "", ask_system_prompt: "" };

export default function AdminSettingsPage() {
  const { t } = useI18n();
  const PROTOCOLS: { value: string; label: string }[] = [
    { value: "openai-chat", label: "OpenAI Chat Completions" },
    { value: "anthropic", label: t("admin.settings.anthropic_messages_native") },
    { value: "gemini", label: t("admin.settings.gemini_generatecontent_native") },
    { value: "openai-responses", label: "OpenAI Responses API" },
  ];
  // Provider presets only fill the API format + Base URL — the model is always
  // fetched from the endpoint, never hardcoded. Both fields stay editable.
  const PRESETS: { label: string; base: string; protocol: string }[] = [
    { label: "OpenAI", base: "https://api.openai.com/v1", protocol: "openai-chat" },
    { label: "Anthropic Claude", base: "https://api.anthropic.com", protocol: "anthropic" },
    { label: "Google Gemini", base: "https://generativelanguage.googleapis.com", protocol: "gemini" },
    { label: "DeepSeek", base: "https://api.deepseek.com/v1", protocol: "openai-chat" },
    { label: t("admin.settings.qwen"), base: "https://dashscope.aliyuncs.com/compatible-mode/v1", protocol: "openai-chat" },
    { label: t("admin.settings.zhipu_glm"), base: "https://open.bigmodel.cn/api/paas/v4", protocol: "openai-chat" },
    { label: "Kimi (Moonshot)", base: "https://api.moonshot.cn/v1", protocol: "openai-chat" },
    { label: "SiliconFlow", base: "https://api.siliconflow.cn/v1", protocol: "openai-chat" },
    { label: t("admin.settings.ollama_local"), base: "http://localhost:11434/v1", protocol: "openai-chat" },
    { label: t("admin.settings.vllm_self_hosted"), base: "http://localhost:8000/v1", protocol: "openai-chat" },
  ];
  const CHUNK_STRATEGIES = [
    { value: "markdown", label: t("admin.settings.markdown_headings_first") },
    { value: "heading", label: t("admin.settings.heading_level") },
    { value: "fixed", label: t("admin.settings.fixed_length") },
    { value: "semantic", label: t("admin.settings.semantic_chunking") },
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
  const [testingConnection, setTestingConnection] = useState<"" | "chat" | "embedding" | "rerank">("");
  const [connectionResult, setConnectionResult] = useState<{ chat?: ModelConnectionTestResult; embedding?: ModelConnectionTestResult; rerank?: ModelConnectionTestResult }>({});
  const [connectionError, setConnectionError] = useState<{ chat?: string; embedding?: string; rerank?: string }>({});

  const protocol = ai.ask_protocol || "openai-chat";

  async function loadModels() {
    if (!ai.ask_base_url) { setError(t("admin.settings.please_enter_the_api_base_url_first")); return; }
    setLoadingModels(true);
    setError("");
    try {
      const r = await fetchModels(ai.ask_base_url, ai.ask_api_key, protocol);
      setModels(r.models || []);
      if (!r.models?.length) setError(t("admin.settings.this_endpoint_did_not_return_a_model_list"));
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

  async function handleConnectionTest(kind: "chat" | "embedding" | "rerank") {
    const base_url = kind === "chat" ? ai.ask_base_url : kind === "embedding" ? ai.embedding_base_url : ai.rerank_base_url;
    const model = kind === "chat" ? ai.ask_model : kind === "embedding" ? ai.embedding_model : ai.rerank_model;
    const api_key = kind === "chat" ? ai.ask_api_key : kind === "embedding" ? ai.embedding_api_key : ai.rerank_api_key;
    setTestingConnection(kind);
    setError("");
    setConnectionResult((prev) => ({ ...prev, [kind]: undefined }));
    setConnectionError((prev) => ({ ...prev, [kind]: undefined }));
    try {
      const result = await testModelConnection({ kind, protocol: ai.ask_protocol, base_url, model, api_key });
      setConnectionResult((prev) => ({ ...prev, [kind]: result }));
    } catch (e) {
      setConnectionError((prev) => ({ ...prev, [kind]: e instanceof Error ? e.message : String(e) }));
    } finally {
      setTestingConnection("");
    }
  }

  return (
    <AdminShell
      title={t("component.adminShell.model_settings")}
      kicker="AI Model"
      description={t("admin.settings.configure_chat_embedding_and_reranking_models_plus_rag")}
      contentClassName="admin-settings-content"
    >
      {error ? <div className="panel badge-danger" style={{ borderRadius: 12 }}>{error}</div> : null}

      <section className="card" style={{ display: "grid", gap: 18 }}>
        <div className="field">
          <label>{t("admin.settings.quick_presets")}</label>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
            {PRESETS.map((p) => (
              <button key={p.label} className="shelf-tab" onClick={() => { setModels([]); setAI({ ...ai, ask_base_url: p.base, ask_protocol: p.protocol, ask_model: "" }); }}>
                {p.label}
              </button>
            ))}
          </div>
          <span className="field-hint">{t("admin.settings.presets_populate_only_the_api_format_and_base")}</span>
        </div>

        <div className="field">
          <label>{t("admin.settings.api_format")}</label>
          <Combobox
            options={PROTOCOLS}
            value={[protocol]}
            onChange={(v) => { setModels([]); setAI({ ...ai, ask_protocol: v[0] || "openai-chat", ask_model: "" }); }}
            multiple={false}
            placeholder={t("admin.settings.select_api_format")}
          />
          <span className="field-hint">{t("admin.settings.protocolHint")} <code className="code-chip">OpenAI Chat Completions</code>。</span>
        </div>

        <div className="field">
          <label>{t("admin.settings.apiBaseUrl")}</label>
          <input value={ai.ask_base_url || ""} placeholder="https://api.openai.com/v1" onChange={(e) => setAI({ ...ai, ask_base_url: e.target.value })} />
          <span className="field-hint">{t("admin.settings.root_url_of_the_service_respective_paths_e")} <code className="code-chip">/chat/completions</code>、<code className="code-chip">/v1/messages</code>）。</span>
        </div>

        <div className="field">
          <label>{t("admin.settings.apiKey")}</label>
          <div style={{ position: "relative" }}>
            <KeyRound size={15} style={{ position: "absolute", left: 12, top: 13, color: "hsl(var(--muted))" }} />
            <input
              type="password"
              style={{ paddingLeft: 34 }}
              value={ai.ask_api_key || ""}
              placeholder={keySet ? t("admin.settings.configured_leave_blank_to_keep_unchanged") : "sk-…"}
              onChange={(e) => setAI({ ...ai, ask_api_key: e.target.value })}
            />
          </div>
          {keySet ? <span className="field-hint" style={{ color: "hsl(var(--accent-strong))" }}>{t("admin.settings.api_key_saved_enter_a_new_value_to")}</span> : null}
        </div>

        <div className="field">
          <label>{t("admin.settings.model")}</label>
          <div style={{ display: "flex", gap: 8, alignItems: "flex-start" }}>
            <div style={{ flex: 1 }}>
              <Combobox
                options={models.map((m) => ({ value: m, label: m }))}
                value={ai.ask_model ? [ai.ask_model] : []}
                onChange={(v) => setAI({ ...ai, ask_model: v[0] || "" })}
                multiple={false}
                allowCreate
                placeholder={models.length ? t("admin.settings.select_model") : t("admin.settings.click_fetch_model_list_right_to_retrieve_models")}
              />
            </div>
            <button className="button" onClick={loadModels} disabled={loadingModels} style={{ flex: "none", height: 42 }}>
              {loadingModels ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} {t("admin.settings.fetch_model_list")}
            </button>
          </div>
          <span className="field-hint">{t("admin.settings.models_are_fetched_in_real_time_from_the")}</span>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
          <button className="button" onClick={() => handleConnectionTest("chat")} disabled={testingConnection === "chat"}>
            {testingConnection === "chat" ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} 测试问答模型
          </button>
          {connectionResult.chat ? (
            <span className="badge badge-success">
              连接成功：endpoint {connectionResult.chat.endpoint}，返回 {connectionResult.chat.sample}
            </span>
          ) : null}
          {connectionError.chat ? <span className="badge badge-danger">测试失败：{connectionError.chat}</span> : null}
        </div>

        <div style={{ display: "flex", gap: 16 }}>
          <div className="field" style={{ flex: 1 }}>
            <label>{t("admin.settings.max_response_tokens_optional")}</label>
            <input
              type="number"
              min={1}
              value={ai.ask_max_tokens ?? ""}
              placeholder={t("admin.settings.default_4096_required_for_anthropic_leave_blank_for")}
              onChange={(e) => setAI({ ...ai, ask_max_tokens: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
            />
            <span className="field-hint">{t("admin.settings.controls_the_maximum_length_of_a_single_response")}</span>
          </div>
          <div className="field" style={{ flex: 1 }}>
            <label>{t("admin.settings.sampling_temperature_optional")}</label>
            <input
              type="number"
              min={0}
              max={2}
              step={0.1}
              value={ai.ask_temperature ?? ""}
              placeholder={t("admin.settings.default_0_2")}
              onChange={(e) => setAI({ ...ai, ask_temperature: e.target.value === "" ? undefined : parseFloat(e.target.value) })}
            />
            <span className="field-hint">{t("admin.settings.lower_values_indicate_higher_confidence_and_better_alignment")}</span>
          </div>
        </div>

        <div className="field">
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 8 }}>
            <label style={{ margin: 0 }}>{t("admin.settings.system_prompt")}</label>
            <button
              type="button"
              className="shelf-tab"
              onClick={() => setAI({ ...ai, ask_system_prompt: promptDefault })}
              disabled={!promptDefault || ai.ask_system_prompt === promptDefault}
              title={t("admin.settings.restore_to_built_in_default_prompt")}
              style={{ display: "inline-flex", alignItems: "center", gap: 6 }}
            >
              <RotateCcw size={13} /> {t("admin.settings.reset_to_default")}
            </button>
          </div>
          <textarea
            style={{ minHeight: 120, padding: "10px 12px", lineHeight: 1.6 }}
            value={ai.ask_system_prompt || ""}
            placeholder={promptDefault || t("admin.settings.leave_blank_to_use_the_built_in_chinese")}
            onChange={(e) => setAI({ ...ai, ask_system_prompt: e.target.value })}
          />
          <span className="field-hint">{t("admin.settings.built_in_default_prompt_pre_filled_modify_as")}</span>
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("admin.settings.embed_model")}</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.embeddingBaseUrl")}</label>
              <input
                value={ai.embedding_base_url || ""}
                placeholder="https://api.example.com/v1"
                onChange={(e) => setAI({ ...ai, embedding_base_url: e.target.value })}
              />
              <span className="field-hint">填写服务根地址，例如 <code className="code-chip">https://api.example.com/v1</code>；实际请求 <code className="code-chip">/embeddings</code></span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.embedding_model")}</label>
              <input
                value={ai.embedding_model || ""}
                placeholder="text-embedding-3-large / bge-m3"
                onChange={(e) => setAI({ ...ai, embedding_model: e.target.value })}
              />
              <span className="field-hint">{t("admin.settings.enter_the_model_name_as_specified_by_the")}</span>
            </div>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.embeddingApiKey")}</label>
              <input
                type="password"
                value={ai.embedding_api_key || ""}
                placeholder={embeddingKeySet ? t("admin.settings.configured_leave_blank_to_keep_unchanged") : "sk-…"}
                onChange={(e) => setAI({ ...ai, embedding_api_key: e.target.value })}
              />
              <span className="field-hint">{t("admin.settings.embeddingApiKeyHint")}</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.embeddingDimensions")}</label>
              <input type="number" value={1024} readOnly disabled />
              <span className="field-hint">{t("admin.settings.embeddingDimensionsHint")}</span>
            </div>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
            <button className="button" onClick={() => handleConnectionTest("embedding")} disabled={testingConnection === "embedding"}>
              {testingConnection === "embedding" ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} 测试嵌入模型
            </button>
            {connectionResult.embedding ? (
              <span className="badge badge-success">
                连接成功：{connectionResult.embedding.dimension} 维，endpoint {connectionResult.embedding.endpoint}
              </span>
            ) : null}
            {connectionError.embedding ? <span className="badge badge-danger">测试失败：{connectionError.embedding}</span> : null}
          </div>
        </div>

        <div style={{ borderTop: "1px solid hsl(var(--border))", paddingTop: 18, display: "grid", gap: 16 }}>
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("admin.settings.reordering_and_recall_testing")}</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.rerankBaseUrl")}</label>
              <input
                value={ai.rerank_base_url || ""}
                placeholder="https://api.example.com/v1"
                onChange={(e) => setAI({ ...ai, rerank_base_url: e.target.value })}
              />
              <span className="field-hint">填写重排序服务根地址；实际请求 <code className="code-chip">/rerank</code>，不要填嵌入模型的 <code className="code-chip">/embeddings</code></span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.rerank_model")}</label>
              <input
                value={ai.rerank_model || ""}
                placeholder="bge-reranker-v2-m3"
                onChange={(e) => setAI({ ...ai, rerank_model: e.target.value })}
              />
            </div>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.rerankApiKey")}</label>
              <input
                type="password"
                value={ai.rerank_api_key || ""}
                placeholder={rerankKeySet ? t("admin.settings.configured_leave_blank_to_keep_unchanged") : "sk-…"}
                onChange={(e) => setAI({ ...ai, rerank_api_key: e.target.value })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.rerank_topk")}</label>
              <input
                type="number"
                min={1}
                value={ai.rerank_top_k ?? ""}
                placeholder="20"
                onChange={(e) => setAI({ ...ai, rerank_top_k: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
            <button className="button" onClick={() => handleConnectionTest("rerank")} disabled={testingConnection === "rerank"}>
              {testingConnection === "rerank" ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} 测试重排序模型
            </button>
            {connectionResult.rerank ? (
              <span className="badge badge-success">
                连接成功：top index {connectionResult.rerank.top_index}，endpoint {connectionResult.rerank.endpoint}
              </span>
            ) : null}
            {connectionError.rerank ? <span className="badge badge-danger">测试失败：{connectionError.rerank}</span> : null}
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.recall_test_query")}</label>
              <input
                value={ai.recall_test_query || ""}
                placeholder={t("admin.settings.how_to_clear_build_cache")}
                onChange={(e) => setAI({ ...ai, recall_test_query: e.target.value })}
              />
              <span className="field-hint">{t("admin.settings.save_a_baseline_question_for_later_comparison_across")}</span>
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.recall_test_topk")}</label>
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
            <label>{t("admin.settings.expected_hit_document_id")}</label>
            <textarea
              style={{ minHeight: 78, padding: "10px 12px", lineHeight: 1.6 }}
              value={ai.recall_test_doc_ids || ""}
              placeholder="DemoModule:latest:guide"
              onChange={(e) => setAI({ ...ai, recall_test_doc_ids: e.target.value })}
            />
            <span className="field-hint">{t("admin.settings.one_doc_id_per_line_subsequent_retrieval_tests")}</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <button className="button" onClick={handleRecallTest} disabled={recallLoading}>
              {recallLoading ? <Loader2 size={15} className="ds-spin" /> : <ListChecks size={15} />} {t("admin.settings.run_recall_test")}
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
                <span className="tag">{t("admin.settings.hit")} {recallResult.actual_doc_ids.filter((id) => recallResult.expected_doc_ids.includes(id)).length}/{recallResult.expected_doc_ids.length}</span>
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
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("admin.settings.segmentation_policy")}</h2>
          <div style={{ display: "flex", gap: 16 }}>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.policies")}</label>
              <Combobox
                options={CHUNK_STRATEGIES}
                value={ai.chunk_strategy ? [ai.chunk_strategy] : []}
                onChange={(v) => setAI({ ...ai, chunk_strategy: v[0] || "" })}
                multiple={false}
                placeholder={t("admin.settings.select_segmentation_strategy")}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.segment_length")}</label>
              <input
                type="number"
                min={1}
                value={ai.chunk_size ?? ""}
                placeholder="800"
                onChange={(e) => setAI({ ...ai, chunk_size: e.target.value === "" ? undefined : Math.max(1, parseInt(e.target.value, 10) || 0) })}
              />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>{t("admin.settings.overlap_length")}</label>
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
            {saving ? <Loader2 size={16} className="ds-spin" /> : ''} {t("admin.settings.save_settings")}
          </button>
          {saved ? <span className="badge badge-success"><Check size={13} /> {t("admin.snippets.saved")}</span> : null}
        </div>
      </section>

      <p className="muted" style={{ fontSize: 12 }}>
        {t("admin.settings.note_the_chat_model_takes_effect_immediately_for")}
      </p>
    </AdminShell>
  );
}
