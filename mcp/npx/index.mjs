#!/usr/bin/env node
// Modex docs MCP server (stdio, zero-dependency).
//
// A thin Model Context Protocol server that proxies tool calls to a running
// Modex backend's HTTP API. Designed to be launched by an MCP client
// (Claude Code, Cursor, ...) over stdio.
//
// Config via environment variables:
//   MODEX_API_BASE_URL   Modex backend base URL (default http://localhost:8671)
//   MODEX_MCP_TOKEN      Bearer token for the backend (matches backend MCP_TOKEN)
// Back-compat aliases: DOCS_API_BASE_URL, MCP_TOKEN.
//
// Protocol notes: only stdout carries JSON-RPC. All diagnostics go to stderr.

import { createInterface } from "node:readline";

const BASE_URL = (
  process.env.MODEX_API_BASE_URL ||
  process.env.DOCS_API_BASE_URL ||
  "http://localhost:8671"
).replace(/\/+$/, "");
const TOKEN = process.env.MODEX_MCP_TOKEN || process.env.MCP_TOKEN || "";

const log = (...a) => console.error("[modex-docs-mcp]", ...a);

const TOOLS = [
  {
    name: "list_modules",
    description: "List documentation modules, optionally filtered by capability-domain category or keyword.",
    inputSchema: {
      type: "object",
      properties: {
        category_id: { type: "string", description: "Capability-domain category id (e.g. standards.tools.version)" },
        keyword: { type: "string", description: "Free-text keyword filter" },
      },
    },
  },
  {
    name: "list_versions",
    description: "List the available documentation versions for one module.",
    inputSchema: {
      type: "object",
      properties: { module_key: { type: "string", description: "Module key" } },
      required: ["module_key"],
    },
  },
  {
    name: "search_docs",
    description: "Search documentation with keyword, semantic, or hybrid mode. Returns ranked pages.",
    inputSchema: {
      type: "object",
      properties: {
        query: { type: "string", description: "Search query" },
        mode: { type: "string", enum: ["keyword", "semantic", "hybrid"], description: "Search mode (default hybrid)" },
        limit: { type: "number", description: "Max results (default 5)" },
        module_key: { type: "string", description: "Restrict to one module" },
        docs_version: { type: "string", description: "Restrict to one docs version" },
      },
      required: ["query"],
    },
  },
  {
    name: "get_doc_page",
    description: "Fetch one full documentation page by its doc_id.",
    inputSchema: {
      type: "object",
      properties: { doc_id: { type: "string", description: "Document id" } },
      required: ["doc_id"],
    },
  },
];

async function api(method, path, body) {
  const headers = { "Content-Type": "application/json" };
  if (TOKEN) headers["Authorization"] = `Bearer ${TOKEN}`;
  const res = await fetch(BASE_URL + path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  if (!res.ok) {
    throw new Error(`backend ${res.status}: ${typeof data === "string" ? data : JSON.stringify(data)}`);
  }
  return data;
}

function resultCount(v) {
  if (Array.isArray(v)) return v.length;
  if (v && typeof v === "object") {
    if (typeof v.total === "number") return v.total;
    if (Array.isArray(v.results)) return v.results.length;
    return 1;
  }
  return v == null ? 0 : 1;
}

async function callTool(name, args = {}) {
  switch (name) {
    case "list_modules": {
      const q = new URLSearchParams();
      if (args.category_id) q.set("category_id", args.category_id);
      if (args.keyword) q.set("keyword", args.keyword);
      const qs = q.toString();
      return api("GET", "/api/modules" + (qs ? `?${qs}` : ""));
    }
    case "list_versions":
      return api("GET", `/api/modules/${encodeURIComponent(args.module_key || "")}/versions`);
    case "search_docs": {
      const body = {
        query: args.query,
        mode: args.mode || "hybrid",
        page: 1,
        page_size: args.limit || 5,
        filters: {
          modules: args.module_key ? [args.module_key] : undefined,
          docs_versions: args.docs_version ? [args.docs_version] : undefined,
        },
      };
      return api("POST", "/api/search", body);
    }
    case "get_doc_page":
      return api("GET", `/api/docs/page/${encodeURIComponent(args.doc_id || "")}`);
    default:
      throw new Error(`unknown tool ${name}`);
  }
}

// Fire-and-forget usage logging so the admin MCP analytics stays populated.
function logMCP(name, args, count) {
  api("POST", "/api/mcp/log", {
    tool_name: name,
    query: typeof args?.query === "string" ? args.query : "",
    input_json: JSON.stringify(args || {}),
    result_count: count,
  }).catch(() => {});
}

async function handle(req) {
  switch (req.method) {
    case "initialize":
      return {
        protocolVersion: "2024-11-05",
        serverInfo: { name: "modex-docs-mcp", version: "0.1.0" },
        capabilities: { tools: {} },
      };
    case "tools/list":
      return { tools: TOOLS };
    case "tools/call": {
      const { name, arguments: args } = req.params || {};
      const data = await callTool(name, args || {});
      logMCP(name, args, resultCount(data));
      return {
        content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
      };
    }
    default:
      throw new Error(`unsupported method ${req.method}`);
  }
}

function send(obj) {
  process.stdout.write(JSON.stringify(obj) + "\n");
}

const rl = createInterface({ input: process.stdin });
rl.on("line", async (line) => {
  const trimmed = line.trim();
  if (!trimmed) return;
  let req;
  try {
    req = JSON.parse(trimmed);
  } catch (e) {
    send({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "parse error" } });
    return;
  }
  // Notifications (no id) require no response.
  const isNotification = req.id === undefined || req.id === null;
  try {
    const result = await handle(req);
    if (!isNotification) send({ jsonrpc: "2.0", id: req.id, result });
  } catch (e) {
    if (!isNotification) {
      send({ jsonrpc: "2.0", id: req.id, error: { code: -32000, message: String(e.message || e) } });
    } else {
      log("notification error:", e.message || e);
    }
  }
});

log(`ready → ${BASE_URL}${TOKEN ? " (authenticated)" : " (no token)"}`);
