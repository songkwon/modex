import { readFile } from "node:fs/promises";
import path from "node:path";
import { MdxContent } from "@/components/mdx/mdx-content";
import { DocToc } from "@/components/doc-toc";
import { DocSourceToggle } from "@/components/doc-source-toggle";
import { getServerI18n } from "@/lib/i18n-server";

async function loadGuide() {
  return readFile(path.join(process.cwd(), "content/modex-guide.mdx"), "utf8");
}

export default async function ModexGuidePage() {
  const source = await loadGuide();
  const { t } = await getServerI18n();
  return (
    <main className="main docs-shell">
      <section className="doc-layout doc-layout--single">
        <aside className="doc-sidebar" />
        <article className="panel prose doc-page">
          <p className="muted text-sm doc-breadcrumb">Modex / Guide</p>
          <div className="doc-title-row">
            <h1 className="doc-title" id="overview">{t("legacy.0cf89cadff1b")}</h1>
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
