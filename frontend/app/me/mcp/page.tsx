export default function McpUsagePage() {
  return (
    <main className="main">
      <section className="panel" style={{ maxWidth: 820, margin: "0 auto" }}>
        <h1 className="text-2xl font-semibold">MCP 使用</h1>
        <p className="muted mt-3">
          Modex 提供 MCP Server，让 AI 工具（如 Claude、Cursor）安全读取文档平台内容。MCP 通过平台 API 访问，不直接读取存储。
        </p>

        <h2 className="mt-6 font-semibold">可用工具</h2>
        <ul className="mt-2 grid gap-1 text-sm muted">
          <li>· <strong>list_modules</strong> — 按分类/关键词列出模块</li>
          <li>· <strong>list_versions</strong> — 列出某模块的版本</li>
          <li>· <strong>search_docs</strong> — keyword / semantic / hybrid 检索</li>
          <li>· <strong>get_doc_page</strong> — 按 doc_id 读取文档正文</li>
        </ul>

        <h2 className="mt-6 font-semibold">启动 MCP Server</h2>
        <pre className="mt-2 overflow-auto rounded-lg p-3 text-xs" style={{ background: "hsl(var(--muted-panel))" }}>
{`cd mcp
DOCS_API_BASE_URL=http://localhost:8671 MCP_TOKEN=dev-token \\
  go run ./cmd/docs-mcp-server`}
        </pre>

        <h2 className="mt-6 font-semibold">JSON-RPC 示例（stdin）</h2>
        <pre className="mt-2 overflow-auto rounded-lg p-3 text-xs" style={{ background: "hsl(var(--muted-panel))" }}>
{`{"jsonrpc":"2.0","id":1,"method":"tools/list"}
{"jsonrpc":"2.0","id":2,"method":"tools/call",
 "params":{"name":"search_docs",
 "arguments":{"query":"构建缓存怎么清理","mode":"hybrid","limit":5}}}`}
        </pre>

        <p className="muted mt-6 text-xs">
          说明：MCP 的 search_docs 与平台搜索共用同一套检索能力。管理员可在「管理 → MCP 日志」查看调用记录。
        </p>
      </section>
    </main>
  );
}
