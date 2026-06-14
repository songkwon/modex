// Mintlify-style reusable content: expand <Snippet name="key"/> partials and
// substitute {{variable}} placeholders before the source reaches compileMDX.

const SNIPPET_RE = /<Snippet\s+[^>]*\bname=["']([^"']+)["'][^>]*\/?>(\s*<\/Snippet>)?/g;
const VAR_RE = /\{\{\s*([\w.-]+)\s*\}\}/g;

export function expandSnippets(
  source: string,
  snippets: Record<string, string>,
  vars: Record<string, string>,
  maxDepth = 6
): string {
  let out = source;
  // Expand snippets repeatedly so nested <Snippet/> references resolve; the
  // depth cap stops runaway/cyclic includes.
  for (let i = 0; i < maxDepth; i++) {
    let replaced = false;
    out = out.replace(SNIPPET_RE, (m, key: string) => {
      if (Object.prototype.hasOwnProperty.call(snippets, key)) {
        replaced = true;
        return snippets[key];
      }
      return m; // unknown name: leave for the <Snippet> fallback component
    });
    if (!replaced) break;
  }
  // Substitute variables (unknown keys are left untouched).
  out = out.replace(VAR_RE, (m, key: string) =>
    Object.prototype.hasOwnProperty.call(vars, key) ? vars[key] : m
  );
  return out;
}
