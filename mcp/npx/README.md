# modex-docs-mcp

Connect any [MCP](https://modelcontextprotocol.io) client (Claude Code, Cursor, Windsurf, …)
to your Modex documentation platform. Zero dependencies, runs over stdio via `npx`.

## Tools

| Tool | Description |
|------|-------------|
| `list_modules` | List doc modules, filter by capability-domain category or keyword |
| `list_versions` | List versions of a module |
| `search_docs` | Keyword / semantic / hybrid search across docs |
| `get_doc_page` | Fetch a full page by `doc_id` |

## Configuration

| Env var | Default | Notes |
|---------|---------|-------|
| `MODEX_API_BASE_URL` | `http://localhost:8671` | Modex backend base URL |
| `MODEX_MCP_TOKEN` | _(none)_ | Bearer token, must match backend `MCP_TOKEN` |

## Install

### Claude Code

```bash
claude mcp add modex-docs \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y modex-docs-mcp
```

### Cursor / Windsurf / generic MCP client

Add to your MCP config (e.g. `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "modex-docs": {
      "command": "npx",
      "args": ["-y", "modex-docs-mcp"],
      "env": {
        "MODEX_API_BASE_URL": "https://modex.example.com",
        "MODEX_MCP_TOKEN": "your-token"
      }
    }
  }
}
```

### Try it locally

```bash
MODEX_API_BASE_URL=http://localhost:8671 MODEX_MCP_TOKEN=dev-token npx -y modex-docs-mcp
```

Then send a JSON-RPC line on stdin, e.g.:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_docs","arguments":{"query":"git 规范"}}}
```
