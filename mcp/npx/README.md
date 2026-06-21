# modex-mcp

Connect any [MCP](https://modelcontextprotocol.io) client (Claude Code, Cursor, Windsurf, …)
to your Modex documentation platform. Zero dependencies, runs over stdio via `npx`.

For hosted deployments, Modex also ships a `modex-mcp-server` container that exposes
streamable HTTP at `/mcp`. Use the `npx` package for clients that only support local
stdio MCP servers.

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
| `MODEX_MCP_TOKEN` | _(none)_ | Personal bearer token generated on the Modex MCP page |

## Install

### Streamable HTTP

Run the MCP image alongside Modex and point compatible MCP clients at:

```text
https://modex.example.com/mcp
```

For local compose deployments with the `mcp` profile enabled:

```text
http://localhost:8787/mcp
```

### Claude Code

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y modex-mcp
```

### Cursor / Windsurf / generic MCP client

Add to your MCP config (e.g. `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "modex": {
      "command": "npx",
      "args": ["-y", "modex-mcp"],
      "env": {
        "MODEX_API_BASE_URL": "https://modex.example.com",
        "MODEX_MCP_TOKEN": "your-token"
      }
    }
  }
}
```

### Modex release tarball

Modex releases can bundle this package and serve it from the platform backend:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y https://modex.example.com/api/mcp/dist/modex-mcp.tgz
```

If your organization keeps the MCP package in an installable Git repository:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y git+https://github.com/your-org/modex-mcp.git
```

### Modex Skill

The optional Modex Skill is installed separately and is intended for Modex users:

```bash
npx skills add https://modex.example.com
```

Git-hosted skill package:

```bash
npx skills add https://github.com/your-org/modex/tree/main/mcp/skill
```

### Try it locally

```bash
MODEX_API_BASE_URL=http://localhost:8671 MODEX_MCP_TOKEN=your-personal-token npx -y modex-mcp
```

Then send a JSON-RPC line on stdin, e.g.:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_docs","arguments":{"query":"git 规范"}}}
```
