"use client";

import { useEffect, useState } from "react";
import { Copy, RefreshCw, Check } from "lucide-react";
import { getMCPToken, rotateMCPToken } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "https://modex.example.com";

export default function McpUsagePage() {
  const { t } = useI18n();
  const [token, setToken] = useState("");
  const [loading, setLoading] = useState(true);
  const [copiedCmd, setCopiedCmd] = useState(false);
  const [copiedJson, setCopiedJson] = useState(false);
  const [copiedOffline, setCopiedOffline] = useState(false);
  const [copiedSkill, setCopiedSkill] = useState(false);

  useEffect(() => {
    getMCPToken()
      .then((r) => setToken(r.mcp_token || ""))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const displayToken = token || t("me.mcp.your_token");

  const claudeCmd = `claude mcp add modex \\
  --env MODEX_API_BASE_URL=${API_BASE} \\
  --env MODEX_MCP_TOKEN=${displayToken} \\
  -- npx -y modex-mcp`;

  const tarballUrl = `${API_BASE}/api/mcp/dist/modex-mcp.tgz`;
  const offlineCmd = `claude mcp add modex \\
  --env MODEX_API_BASE_URL=${API_BASE} \\
  --env MODEX_MCP_TOKEN=${displayToken} \\
  -- npx -y ${tarballUrl}`;
  const gitCmd = `claude mcp add modex \\
  --env MODEX_API_BASE_URL=${API_BASE} \\
  --env MODEX_MCP_TOKEN=${displayToken} \\
  -- npx -y git+https://github.com/your-org/modex-mcp.git`;
  const skillCmd = `npx skills add ${API_BASE}`;

  const cursorJson = `{
  "mcpServers": {
    "modex": {
      "command": "npx",
      "args": ["-y", "modex-mcp"],
      "env": {
        "MODEX_API_BASE_URL": "${API_BASE}",
        "MODEX_MCP_TOKEN": "${displayToken}"
      }
    }
  }
}`;

  async function handleRotate() {
    try {
      const r = await rotateMCPToken();
      setToken(r.mcp_token);
    } catch (e) {
      alert(String(e));
    }
  }

  function copyText(text: string, setter: (v: boolean) => void) {
    navigator.clipboard.writeText(text).then(() => {
      setter(true);
      setTimeout(() => setter(false), 1500);
    });
  }

  return (
    <main className="main">
      <section className="grid" style={{ maxWidth: 860, margin: "0 auto" }}>
        <header className="hero-panel">
          <div className="page-kicker">{t("me.mcp.ai_tool_integration")}</div>
          <h1 className="page-title">{t("me.mcp.integrate_modex_into_your_ai_workspace")}</h1>
          <p className="hero-copy">
            {t("me.mcp.this_mcp_and_skill_are_designed_for_modex")}
            <code className="code-chip" style={{ margin: "0 4px" }}>npx</code> {t("me.mcp.launched_on_demand_locally_and_connects_to_the")}
          </p>
        </header>

        <section className="card">
          <div className="flex items-center justify-between" style={{ marginBottom: 12 }}>
            <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("me.mcp.your_mcp_token")}</h2>
            <button className="button button-primary" onClick={handleRotate} disabled={loading}>
              <RefreshCw size={14} /> {token ? t("admin.modules.regenerate") : t("me.mcp.generate_token")}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 13, marginBottom: 10 }}>
            {t("me.mcp.each_user_has_a_unique_token_the_server")}
          </p>
          <div className="mcp-code" style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
            <code style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {loading ? t("component.docReadStats.loading") : displayToken}
            </code>
            {token && (
              <button
                className="icon-btn"
                onClick={() => copyText(token, setCopiedCmd)}
                aria-label={t("me.mcp.copy_token")}
                title={t("me.mcp.copy_token")}
              >
                {copiedCmd ? <Check size={14} /> : <Copy size={14} />}
              </button>
            )}
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>Claude Code</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>{t("me.mcp.run_in_terminal_token_is_auto_filled_with")}</p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{claudeCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(claudeCmd, setCopiedCmd)}
              aria-label={t("me.mcp.copy_command")}
            >
              {copiedCmd ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("me.mcp.cursor_windsurf_etc")}</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>{t("me.mcp.add_to_mcp_configuration_file_e_g")} <code className="code-chip">~/.cursor/mcp.json</code>）：</p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{cursorJson}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(cursorJson, setCopiedJson)}
              aria-label={t("me.mcp.copy_configuration")}
            >
              {copiedJson ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("me.mcp.offline_intranet_installation_no_public_npm_required")}</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            {t("me.mcp.mcp_packages_are_released_alongside_modex_and_also")}
          </p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{offlineCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(offlineCmd, setCopiedOffline)}
              aria-label={t("me.mcp.copy_command")}
            >
              {copiedOffline ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            {t("me.mcp.also_supported_if_you_split_your_mcp_packages")} <code className="code-chip">npx</code> {t("me.mcp.install_directly_from_git")}
          </p>
          <pre className="mcp-code" style={{ marginTop: 8 }}>{gitCmd}</pre>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            {t("me.mcp.or_download_the_tool_output_directly")}
            <a href={tarballUrl} className="code-chip" style={{ margin: "0 6px" }}>modex-mcp.tgz</a>
            {t("me.mcp.also_available_for_download")}
            <a href={`${API_BASE}/api/mcp/dist/index.mjs`} className="code-chip" style={{ margin: "0 6px" }}>index.mjs</a>
            用 <code className="code-chip">node index.mjs</code> {t("me.mcp.run_directly")}
          </p>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("me.mcp.install_modex_skill")}</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            {t("me.mcp.skills_provide_modex_usage_guidelines_retrieval_preferences_and")}
          </p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{skillCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(skillCmd, setCopiedSkill)}
              aria-label={t("me.mcp.copy_command")}
            >
              {copiedSkill ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            {t("me.mcp.also_supported_if_the_skill_is_maintained_in")} <code className="code-chip">npx skills add https://github.com/your-org/modex/tree/main/mcp/skill</code>。
          </p>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("me.mcp.available_tools")}</h2>
          <ul className="mcp-tool-list" style={{ marginTop: 10 }}>
            <li>
              <span className="tag">list_modules</span>
              <span className="mcp-tool-desc">{t("me.mcp.list_document_sources_by_category_keyword")}</span>
            </li>
            <li>
              <span className="tag">list_versions</span>
              <span className="mcp-tool-desc">{t("me.mcp.list_versions_of_a_document_source")}</span>
            </li>
            <li>
              <span className="tag">search_docs</span>
              <span className="mcp-tool-desc">{t("me.mcp.keyword_semantic_hybrid_search")}</span>
            </li>
            <li>
              <span className="tag">get_doc_page</span>
              <span className="mcp-tool-desc">{t("me.mcp.read_document_body_by_doc_id")}</span>
            </li>
          </ul>
          <p className="muted" style={{ fontSize: 12, marginTop: 14 }}>
            {t("me.mcp.note_mcp_shares_the_same_search_capability_with")}
          </p>
        </section>
      </section>
    </main>
  );
}
