import { readFile } from "node:fs/promises";
import path from "node:path";
import { MdxContent } from "@/components/mdx/mdx-content";
import { DocToc } from "@/components/doc-toc";
import { DocFooter } from "@/components/doc-footer";
import { DocSourceToggle } from "@/components/doc-source-toggle";

const fallback = `# Markdown 示例文档

这篇文档是 Modex 文档系统的写作样例。支持标准 Markdown、GFM、数学公式、Mermaid、Kroki/PlantUML、Mintlify 风格 MDX 组件、上传插件组件和多平台代码块。

## 多平台代码块

<CodeGroup>

\`\`\`bash npm
npm install @modex/docs
\`\`\`

\`\`\`bash pnpm
pnpm add @modex/docs
\`\`\`

\`\`\`bash bun
bun add @modex/docs
\`\`\`

</CodeGroup>

## UML 和图表

\`\`\`mermaid
graph LR
  A[Markdown] --> B[Modex]
  B --> C[Search]
  C --> D[MCP]
\`\`\`

## 数学公式

行内公式：$T = hT_c + (1-h)T_s$。

$$
QPS = \\frac{requests}{seconds}
$$
`;

async function loadShowcase() {
  const candidates = [
    path.join(process.cwd(), "content/markdown-showcase.md"),
    path.join(process.cwd(), "../backend/internal/store/seeddata/markdown-showcase.md"),
  ];
  for (const candidate of candidates) {
    try {
      return await readFile(candidate, "utf8");
    } catch {
      // Try the next location; standalone frontend builds do not include backend files.
    }
  }
  return fallback;
}

export default async function DocsExamplePage() {
  const source = await loadShowcase();
  return (
    <main className="main docs-shell">
      <section className="doc-layout doc-layout--single">
        <aside className="doc-sidebar" />
        <article className="panel prose doc-page">
          <p className="muted text-sm doc-breadcrumb">Modex / 写作参考</p>
          <div className="doc-title-row">
            <h1 className="doc-title" id="overview">Markdown 示例文档</h1>
          </div>
          <DocSourceToggle source={source}>
            <MdxContent source={source} />
          </DocSourceToggle>
          <DocFooter docId="modex:example:markdown-showcase" />
        </article>
        <aside className="doc-toc">
          <div className="doc-toc-card">
            <DocToc />
          </div>
        </aside>
      </section>
    </main>
  );
}
