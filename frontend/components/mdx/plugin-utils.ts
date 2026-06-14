// Plain (server-safe) helpers for uploaded plugins. Kept out of the "use client"
// sandbox module so the server MdxContent can call them during render.

// Keep only JSON-serializable attribute props. Uploaded component plugins receive
// attributes, not rich JSX children — children are dropped before crossing the
// sandbox boundary.
export function serializableProps(p: Record<string, unknown> | undefined): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(p || {})) {
    if (k === "children") continue;
    const t = typeof v;
    if (t === "string" || t === "number" || t === "boolean") out[k] = v;
  }
  return out;
}
