"use client";

import { useEffect, useState } from "react";
import { Copy, RefreshCw, Check } from "lucide-react";
import { getMCPToken, rotateMCPToken } from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "https://modex.example.com";

export default function McpUsagePage() {
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

  const displayToken = token || "<你的 token>";

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
          <div className="page-kicker">AI 工具接入</div>
          <h1 className="page-title">把 Modex 接入你的 AI 工作台</h1>
          <p className="hero-copy">
            这里的 MCP 和 Skill 面向 Modex 使用者。配置后，Claude Code / Cursor 等工具可以检索并阅读本平台文档。MCP 工具通过
            <code className="code-chip" style={{ margin: "0 4px" }}>npx</code> 在本地按需拉起，连接到平台后端 API。
          </p>
        </header>

        <section className="card">
          <div className="flex items-center justify-between" style={{ marginBottom: 12 }}>
            <h2 style={{ fontSize: 16, fontWeight: 720 }}>你的 MCP Token</h2>
            <button className="button button-primary" onClick={handleRotate} disabled={loading}>
              <RefreshCw size={14} /> {token ? "重新生成" : "生成 Token"}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 13, marginBottom: 10 }}>
            每个用户拥有独立 token，服务端可在 MCP 日志中追踪调用来源。
          </p>
          <div className="mcp-code" style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
            <code style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {loading ? "加载中…" : displayToken}
            </code>
            {token && (
              <button
                className="icon-btn"
                onClick={() => copyText(token, setCopiedCmd)}
                aria-label="复制 token"
                title="复制 token"
              >
                {copiedCmd ? <Check size={14} /> : <Copy size={14} />}
              </button>
            )}
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>Claude Code</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>在终端执行（token 已自动填入上方生成的值）：</p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{claudeCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(claudeCmd, setCopiedCmd)}
              aria-label="复制命令"
            >
              {copiedCmd ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>Cursor / Windsurf 等</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>加入 MCP 配置文件（如 <code className="code-chip">~/.cursor/mcp.json</code>）：</p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{cursorJson}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(cursorJson, setCopiedJson)}
              aria-label="复制配置"
            >
              {copiedJson ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>离线 / 内网安装（无需公网 npm）</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            MCP 包会随 Modex release 一起发布，也会由本平台后端提供下载。适合内网环境，无需访问 npmjs.com：
          </p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{offlineCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(offlineCmd, setCopiedOffline)}
              aria-label="复制命令"
            >
              {copiedOffline ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            如果你们把 MCP 包拆到独立 Git 仓库，也可以让 <code className="code-chip">npx</code> 直接从 Git 安装：
          </p>
          <pre className="mcp-code" style={{ marginTop: 8 }}>{gitCmd}</pre>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            或直接下载工具产物：
            <a href={tarballUrl} className="code-chip" style={{ margin: "0 6px" }}>modex-mcp.tgz</a>
            （也可下载
            <a href={`${API_BASE}/api/mcp/dist/index.mjs`} className="code-chip" style={{ margin: "0 6px" }}>index.mjs</a>
            用 <code className="code-chip">node index.mjs</code> 直接运行）。
          </p>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>安装 Modex Skill</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            Skill 用于给支持 skills 的 AI 客户端补充 Modex 使用规范、检索偏好和回答约束；MCP 负责真正读取平台文档。
          </p>
          <div style={{ position: "relative" }}>
            <pre className="mcp-code">{skillCmd}</pre>
            <button
              className="icon-btn"
              style={{ position: "absolute", top: 8, right: 8, background: "hsl(var(--panel))" }}
              onClick={() => copyText(skillCmd, setCopiedSkill)}
              aria-label="复制命令"
            >
              {copiedSkill ? <Check size={14} /> : <Copy size={14} />}
            </button>
          </div>
          <p className="muted" style={{ fontSize: 12, marginTop: 10 }}>
            如果 Skill 在 Git 仓库维护，也可以使用 <code className="code-chip">npx skills add https://github.com/your-org/modex/tree/main/mcp/skill</code>。
          </p>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>可用工具</h2>
          <ul className="mcp-tool-list" style={{ marginTop: 10 }}>
            <li>
              <span className="tag">list_modules</span>
              <span className="mcp-tool-desc">按分类 / 关键词列出文档源</span>
            </li>
            <li>
              <span className="tag">list_versions</span>
              <span className="mcp-tool-desc">列出某文档源的版本</span>
            </li>
            <li>
              <span className="tag">search_docs</span>
              <span className="mcp-tool-desc">keyword / semantic / hybrid 检索</span>
            </li>
            <li>
              <span className="tag">get_doc_page</span>
              <span className="mcp-tool-desc">按 doc_id 读取文档正文</span>
            </li>
          </ul>
          <p className="muted" style={{ fontSize: 12, marginTop: 14 }}>
            说明：MCP 与平台搜索共用同一套检索能力。Skill 只提供客户端侧使用规范，不保存平台数据。
          </p>
        </section>
      </section>
    </main>
  );
}
