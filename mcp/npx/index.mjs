#!/usr/bin/env node

import { McpServer, ResourceTemplate } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import * as z from "zod/v4";

const BASE_URL = (
  process.env.MODEX_API_BASE_URL ||
  process.env.DOCS_API_BASE_URL ||
  "http://localhost:8671"
).replace(/\/+$/, "");
const TOKEN = process.env.MODEX_MCP_TOKEN || "";

const server = new McpServer({ name: "modex-mcp", version: "0.1.0" });
const objectValue = z.record(z.string(), z.unknown());

async function api(method, path, body) {
  const headers = { "Content-Type": "application/json" };
  if (TOKEN) headers.Authorization = `Bearer ${TOKEN}`;
  const response = await fetch(BASE_URL + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  let data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  if (!response.ok) {
    const detail = typeof data === "string" ? data : JSON.stringify(data);
    throw new Error(`backend ${response.status}: ${detail}`);
  }
  return data;
}

function resultCount(value) {
  if (Array.isArray(value)) return value.length;
  if (value && typeof value === "object") {
    if (typeof value.total === "number") return value.total;
    if (Array.isArray(value.results)) return value.results.length;
    return 1;
  }
  return value == null ? 0 : 1;
}

function registerTool(name, config, handler) {
  server.registerTool(name, config, async args => {
    const result = await handler(args);
    api("POST", "/api/mcp/log", {
      tool_name: name,
      query: typeof args.query === "string" ? args.query : "",
      input_json: JSON.stringify(args),
      result_count: resultCount(Object.values(result.structuredContent)[0]),
    }).catch(() => {});
    return result;
  });
}

function structured(data) {
  return {
    content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    structuredContent: data,
  };
}

registerTool("list_modules", {
  description: "List documentation modules by category or keyword.",
  inputSchema: {
    category_id: z.string().optional().describe("Optional category id."),
    keyword: z.string().optional().describe("Optional module keyword."),
  },
  outputSchema: { modules: z.array(objectValue) },
}, async ({ category_id, keyword }) => {
  const query = new URLSearchParams();
  if (category_id) query.set("category_id", category_id);
  if (keyword) query.set("keyword", keyword);
  const suffix = query.size ? `?${query}` : "";
  return structured({ modules: await api("GET", `/api/modules${suffix}`) });
});

registerTool("list_versions", {
  description: "List available documentation versions for one module.",
  inputSchema: { module_key: z.string().min(1).describe("Module key.") },
  outputSchema: { versions: z.array(objectValue) },
}, async ({ module_key }) => structured({
  versions: await api("GET", `/api/modules/${encodeURIComponent(module_key)}/versions`),
}));

registerTool("search_docs", {
  description: "Search documentation with keyword, semantic, or hybrid mode.",
  inputSchema: {
    query: z.string().min(1).describe("Search query."),
    mode: z.enum(["keyword", "semantic", "hybrid"]).optional(),
    limit: z.number().int().min(1).max(20).optional(),
    module_key: z.string().optional(),
    docs_version: z.string().optional(),
  },
  outputSchema: { search: objectValue },
}, async args => structured({
  search: await api("POST", "/api/search", {
    query: args.query,
    mode: args.mode || "hybrid",
    page: 1,
    page_size: args.limit || 5,
    filters: {
      modules: args.module_key ? [args.module_key] : undefined,
      docs_versions: args.docs_version ? [args.docs_version] : undefined,
    },
  }),
}));

registerTool("get_doc_page", {
  description: "Fetch one full documentation page by its doc_id.",
  inputSchema: { doc_id: z.string().min(1).describe("Document id returned by search_docs.") },
  outputSchema: { page: objectValue },
}, async ({ doc_id }) => structured({
  page: await api("GET", `/api/docs/page/${encodeURIComponent(doc_id)}`),
}));

server.registerResource(
  "modex_document",
  new ResourceTemplate("modex://docs/{doc_id}", { list: undefined }),
  {
    title: "Modex document",
    description: "Read a Modex documentation page by doc_id.",
    mimeType: "text/markdown",
  },
  async (uri, { doc_id }) => {
    const page = await api("GET", `/api/docs/page/${encodeURIComponent(String(doc_id))}`);
    const text = page.content_md || page.content_text || JSON.stringify(page, null, 2);
    return {
      contents: [{
        uri: uri.href,
        mimeType: page.content_md || page.content_text ? "text/markdown" : "application/json",
        text,
      }],
    };
  },
);

await server.connect(new StdioServerTransport());
console.error(`[modex-mcp] ready -> ${BASE_URL}${TOKEN ? " (authenticated)" : " (no token)"}`);
