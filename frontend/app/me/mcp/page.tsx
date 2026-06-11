const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "https://modex.example.com";

export default function McpUsagePage() {
  const claudeCmd = `claude mcp add modex-docs \\
  --env MODEX_API_BASE_URL=${API_BASE} \\
  --env MODEX_MCP_TOKEN=<你的 token> \\
  -- npx -y modex-docs-mcp`;

  const cursorJson = `{
  "mcpServers": {
    "modex-docs": {
      "command": "npx",
      "args": ["-y", "modex-docs-mcp"],
      "env": {
        "MODEX_API_BASE_URL": "${API_BASE}",
        "MODEX_MCP_TOKEN": "<你的 token>"
      }
    }
  }
}`;

  return (
    <main className="main">
      <section className="grid" style={{ maxWidth: 860, margin: "0 auto" }}>
        <header className="hero-panel">
          <div className="page-kicker">MCP · AI 接入</div>
          <h1 className="page-title">把 Modex 文档接入你的 AI</h1>
          <p className="hero-copy">
            一行命令即可让 Claude Code / Cursor 等工具检索并阅读本平台文档。无需自己启动服务 —— MCP 工具通过
            <code className="code-chip" style={{ margin: "0 4px" }}>npx</code> 在本地按需拉起，连接到平台后端 API。
          </p>
        </header>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>Claude Code</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>在终端执行（把 token 换成你的）：</p>
          <pre className="mcp-code">{claudeCmd}</pre>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>Cursor / Windsurf 等</h2>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>加入 MCP 配置文件（如 <code className="code-chip">~/.cursor/mcp.json</code>）：</p>
          <pre className="mcp-code">{cursorJson}</pre>
        </section>

        <section className="card">
          <h2 style={{ fontSize: 16, fontWeight: 720 }}>可用工具</h2>
          <ul style={{ marginTop: 10, display: "grid", gap: 8, fontSize: 14 }}>
            <li><span className="tag">list_modules</span> 按分类 / 关键词列出文档源</li>
            <li><span className="tag">list_versions</span> 列出某文档源的版本</li>
            <li><span className="tag">search_docs</span> keyword / semantic / hybrid 检索</li>
            <li><span className="tag">get_doc_page</span> 按 doc_id 读取文档正文</li>
          </ul>
          <p className="muted" style={{ fontSize: 12, marginTop: 14 }}>
            说明：MCP 与平台搜索共用同一套检索能力。token 即平台的 <code className="code-chip">MCP_TOKEN</code>；管理员可在「管理 → MCP 日志」查看调用记录。
          </p>
        </section>
      </section>
    </main>
  );
}
