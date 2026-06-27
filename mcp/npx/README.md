# modex-mcp

Connect any [MCP](https://modelcontextprotocol.io) client (Claude Code, Cursor, Windsurf, …)
to your Modex documentation platform. It uses the official MCP TypeScript SDK
and runs over stdio via `npx`.

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

All tools return MCP structured content with an output schema. Documents are
also available through the `modex://docs/{doc_id}` resource template.

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

The HTTP endpoint requires a valid OAuth access token or personal MCP token.
Authentication, scope checks, and `WWW-Authenticate` challenges are handled by
the official MCP Go SDK.

### Codex OAuth

Codex supports streamable HTTP MCP servers with OAuth. Modex ships a built-in
public OAuth client for Codex named `codex-cli`, so users do not need to copy a
personal MCP token into their Codex config.

Hosted deployment:

```bash
codex mcp add modex \
  --url https://modex.example.com/mcp \
  --oauth-client-id codex-cli

codex mcp login modex --scopes modex:mcp:read,modex:docs:read
```

Local compose deployment with the `mcp` profile:

```bash
codex mcp add modex \
  --url http://localhost:8787/mcp \
  --oauth-client-id codex-cli

codex mcp login modex --scopes modex:mcp:read,modex:docs:read
```

During login, Codex opens a browser, Modex redirects through your OIDC login if
needed, and Codex stores the resulting MCP OAuth credentials in its own
credential store. If Codex uses a fixed callback port, set
`mcp_oauth_callback_port` in `~/.codex/config.toml`; Modex allows loopback
callbacks for `localhost`, `127.0.0.1`, and `::1`.

For hosted deployments behind Nginx or another reverse proxy, expose `/mcp`,
`/.well-known/oauth-protected-resource`, and
`/.well-known/oauth-protected-resource/mcp` to the `modex-mcp-server`. Expose
`/oauth/*` to the Modex backend. OAuth metadata derives the public site URL
from the request host and standard proxy forwarding headers.

### Claude Code

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y https://modex.example.com/api/mcp/dist/modex-mcp.tgz
```

### Cursor / Windsurf / generic MCP client

Add to your MCP config (e.g. `~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "modex": {
      "command": "npx",
      "args": ["-y", "https://modex.example.com/api/mcp/dist/modex-mcp.tgz"],
      "env": {
        "MODEX_API_BASE_URL": "https://modex.example.com",
        "MODEX_MCP_TOKEN": "your-token"
      }
    }
  }
}
```

The tarball is served by the Modex backend at `/api/mcp/dist/modex-mcp.tgz`.

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
MODEX_API_BASE_URL=http://localhost:8671 MODEX_MCP_TOKEN=your-personal-token npx -y http://localhost:8671/api/mcp/dist/modex-mcp.tgz
```

Then send a JSON-RPC line on stdin, e.g.:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_docs","arguments":{"query":"git 规范"}}}
```
