---
name: modex
description: Use Modex MCP to search and read a team's live Modex documentation portal before answering module, API, release, architecture, or platform-specific questions.
---

# Modex

Use this skill when answering questions with a Modex documentation portal.

## Behavior

- Prefer the Modex MCP server for live documentation lookup.
- Search first when the answer depends on current project docs, module versions, or platform-specific rules.
- Cite document titles or paths when the MCP result includes them.
- Do not guess missing platform facts; say that the answer was not found in Modex.
- Keep answers concise and action-oriented for Modex users.

## MCP

For hosted Modex deployments, prefer the streamable HTTP MCP endpoint when the client supports it:

```text
https://modex.example.com/mcp
```

For clients that only support launching a local stdio MCP server, install the package served by Modex with:

```bash
claude mcp add modex \
  --env MODEX_API_BASE_URL=https://modex.example.com \
  --env MODEX_MCP_TOKEN=your-token \
  -- npx -y https://modex.example.com/api/mcp/dist/modex-mcp.tgz
```

The MCP server exposes:

- `list_modules`
- `list_versions`
- `search_docs`
- `get_doc_page`
