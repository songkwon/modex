import { readFile } from "node:fs/promises";
import path from "node:path";
import { MdxContent } from "@/components/mdx/mdx-content";
import { DocToc } from "@/components/doc-toc";
import { DocSourceToggle } from "@/components/doc-source-toggle";

async function loadGuide() {
  return readFile(path.join(process.cwd(), "content/modex-guide.mdx"), "utf8");
}

export default async function ModexGuidePage() {
  const source = await loadGuide();
  return (
    <main className="main docs-shell">
      <section className="doc-layout doc-layout--single">
        <aside className="doc-sidebar" />
        <article className="panel prose doc-page">
          <p className="muted text-sm doc-breadcrumb">Modex / Guide</p>
          <div className="doc-title-row">
            <h1 className="doc-title" id="overview">Modex 项目指南</h1>
          </div>
          <DocSourceToggle source={source}>
            <MdxContent source={source} />
          </DocSourceToggle>
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
