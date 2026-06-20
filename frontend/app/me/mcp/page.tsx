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

  const displayToken = token || t("legacy.c3725c6f9ffa");

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
          <div className="page-kicker">{t("legacy.96ea668f0340")}</div>
          <h1 className="page-title">{t("legacy.e5d0b332a643")}</h1>
          <p className="hero-copy">
            {t("legacy.41cb16b7f958")}
            <code className="code-chip" style={{ margin: "0 4px" }}>npx</code> {t("legacy.5cf4f6c8999e")}
          </p>
        </header>

        <section className="card">
          <div className="flex items-center justify-between" style={{ marginBottom: 12 }}>
            <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.1944022afe3d")}</h2>
            <button className="button button-primary" onClick={handleRotate} disabled={loading}>
              <RefreshCw size={14} /> {token ? t("legacy.3221a042ea36") : t("legacy.f9e0988907b1")}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 13, marginBottom: 10 }}>
            {t("legacy.99da96d9cc60")}
          </p>
          <div className="mcp-code" style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
            <code style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {loading ? t("legacy.4927a53bcc88") : displayToken}
            </code>
            {token && (
              <button
                className="icon-btn"
                onClick={() => copyText(token, setCopiedCmd)}
                aria-label={t("legacy.9541f4a07849")}
                title={t("legacy.9541f4a07849")}
              >
                {copiedCmd ? <Check size={14} /> : <Copy size={14} />}
              </button>
            )}
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>Claude Code</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>{t("legacy.c3e840cc5817")}</p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{claudeCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(claudeCmd, setCopiedCmd)}
              aria-label={t("legacy.605c82dd96bc")}
            >
              {copiedCmd ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.d4db4558fdd4")}</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>{t("legacy.aa141ab924f9")} <code className="code-chip">~/.cursor/mcp.json</code>）：</p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{cursorJson}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(cursorJson, setCopiedJson)}
              aria-label={t("legacy.6bf7933df3bb")}
            >
              {copiedJson ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.0b57f8fcc9d2")}</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            {t("legacy.45a14bc557f9")}
          </p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{offlineCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(offlineCmd, setCopiedOffline)}
              aria-label={t("legacy.605c82dd96bc")}
            >
              {copiedOffline ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            {t("legacy.fc8479135d5b")} <code className="code-chip">npx</code> {t("legacy.2334c248bed2")}
          </p>
          <pre className="mcp-code" style={{ marginTop: 8 }}>{gitCmd}</pre>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            {t("legacy.45cba98c6caf")}
            <a href={tarballUrl} className="code-chip" style={{ margin: "0 6px" }}>modex-mcp.tgz</a>
            {t("legacy.5a2c57dc4d25")}
            <a href={`${API_BASE}/api/mcp/dist/index.mjs`} className="code-chip" style={{ margin: "0 6px" }}>index.mjs</a>
            用 <code className="code-chip">node index.mjs</code> {t("legacy.c21137d45ca1")}
          </p>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.3e8ad9fe8b24")}</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            {t("legacy.73f21d4d035f")}
          </p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{skillCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(skillCmd, setCopiedSkill)}
              aria-label={t("legacy.605c82dd96bc")}
            >
              {copiedSkill ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            {t("legacy.52776e43b2c6")} <code className="code-chip">npx skills add https://github.com/your-org/modex/tree/main/mcp/skill</code>。
          </p>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>{t("legacy.d249cdf8bc64")}</h2>
          <ul className="mcp-tool-list" style={{ marginTop: 10 }}>
            <li>
              <span className="tag">list_modules</span>
              <span className="mcp-tool-desc">{t("legacy.b31ee4dade3a")}</span>
            </li>
            <li>
              <span className="tag">list_versions</span>
              <span className="mcp-tool-desc">{t("legacy.127f8a6a8026")}</span>
            </li>
            <li>
              <span className="tag">search_docs</span>
              <span className="mcp-tool-desc">{t("legacy.34ae19c383c7")}</span>
            </li>
            <li>
              <span className="tag">get_doc_page</span>
              <span className="mcp-tool-desc">{t("legacy.0d1c3a645729")}</span>
            </li>
          </ul>
          <p className="muted" style={{ fontSize: 12, marginTop: 14 }}>
            {t("legacy.86bcd08e4b03")}
          </p>
        </section>
      </section>
    </main>
  );
}
