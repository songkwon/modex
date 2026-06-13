// Custom remark/rehype plugins layered on top of remark-gfm for the doc engine.

// Carry a fenced code block's "meta" (e.g. ```ts config.ts) onto the <code>
// element so the renderer can show a filename header.
export function remarkCodeMeta() {
  const walk = (node: any) => {
    if (!node || typeof node !== "object") return;
    if (node.type === "code" && node.meta) {
      node.data = node.data || {};
      node.data.hProperties = { ...(node.data.hProperties || {}), "data-meta": node.meta };
    }
    if (Array.isArray(node.children)) node.children.forEach(walk);
  };
  return (tree: any) => walk(tree);
}

// GitHub-style alerts: `> [!NOTE]` blockquotes become our callout components.
// IMPORTANT → info, CAUTION → danger; the rest map to the same-named callout.
const ALERT_MAP: Record<string, string> = {
  NOTE: "Note",
  TIP: "Tip",
  IMPORTANT: "Info",
  WARNING: "Warning",
  CAUTION: "Danger"
};

export function remarkGithubAlerts() {
  return (tree: any) => {
    const visit = (node: any) => {
      if (!node || !Array.isArray(node.children)) return;
      node.children.forEach((child: any, i: number) => {
        if (child.type === "blockquote" && child.children?.[0]?.type === "paragraph") {
          const para = child.children[0];
          const first = para.children?.[0];
          if (first?.type === "text") {
            const m = first.value.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*\n?/i);
            if (m) {
              const name = ALERT_MAP[m[1].toUpperCase()];
              first.value = first.value.slice(m[0].length);
              // Drop a leading hard-break left over from the marker line.
              if (first.value === "" && para.children[1]?.type === "break") {
                para.children.splice(0, 2);
              } else if (first.value === "") {
                para.children.splice(0, 1);
              }
              if (para.children.length === 0) child.children.shift();
              node.children[i] = {
                type: "mdxJsxFlowElement",
                name,
                attributes: [],
                children: child.children
              };
            }
          }
        }
        visit(child);
      });
    };
    visit(tree);
  };
}

// Auto table-of-contents: a paragraph containing only `[[toc]]` or `[toc]` is
// replaced (after rehype-slug assigns ids) with a nested nav of h2–h4 links.
export function rehypeToc() {
  return (tree: any) => {
    const headings: { depth: number; id: string; text: string }[] = [];
    const textOf = (n: any): string =>
      n.type === "text" ? n.value : (n.children || []).map(textOf).join("");

    const collect = (node: any) => {
      if (node.type === "element" && /^h[2-4]$/.test(node.tagName) && node.properties?.id) {
        headings.push({ depth: Number(node.tagName[1]), id: String(node.properties.id), text: textOf(node) });
      }
      (node.children || []).forEach(collect);
    };
    collect(tree);
    if (headings.length === 0) return;

    const isMarker = (node: any) => {
      if (node.type !== "element" || node.tagName !== "p" || node.children?.length !== 1) return false;
      const t = node.children[0];
      return t.type === "text" && /^\[\[?toc\]?\]$/i.test(t.value.trim());
    };

    const buildList = (): any => ({
      type: "element",
      tagName: "ol",
      properties: {},
      children: headings.map((h) => ({
        type: "element",
        tagName: "li",
        properties: { className: [`mdx-toc__l${h.depth}`] },
        children: [
          {
            type: "element",
            tagName: "a",
            properties: { href: `#${h.id}` },
            children: [{ type: "text", value: h.text }]
          }
        ]
      }))
    });

    const replace = (node: any) => {
      if (!Array.isArray(node.children)) return;
      node.children = node.children.map((c: any) => {
        if (isMarker(c)) {
          return { type: "element", tagName: "nav", properties: { className: ["mdx-toc"] }, children: [buildList()] };
        }
        replace(c);
        return c;
      });
    };
    replace(tree);
  };
}
