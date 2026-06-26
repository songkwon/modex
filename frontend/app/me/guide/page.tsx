import { readFile } from "node:fs/promises";
import path from "node:path";
import { MdxContent } from "@/components/mdx/mdx-content";
import { DocToc } from "@/components/doc-toc";
import { DocSourceToggle } from "@/components/doc-source-toggle";
import { getServerI18n } from "@/lib/i18n-server";
import { publicApiBaseURL, publicAppTitle, publicDocsctlURL } from "@/lib/runtime-config";

async function loadGuide() {
  return readFile(path.join(process.cwd(), "content/modex-guide.mdx"), "utf8");
}

export default async function ModexGuidePage() {
  const apiBase = publicApiBaseURL();
  const appTitle = publicAppTitle();
  const deployURL = `${apiBase.replace(/\/+$/, "")}/api/deploy`;
  const source = (await loadGuide())
    .replaceAll("{{APP_TITLE}}", appTitle)
    .replaceAll("{{MODEX_DEPLOY_URL}}", deployURL)
    .replaceAll("{{MODEX_DOCSCTL_URL}}", publicDocsctlURL());
  const { t } = await getServerI18n();
  return (
    <main className="main docs-shell">
      <section className="doc-layout doc-layout--single">
        <aside className="doc-sidebar" />
        <article className="panel prose doc-page">
          <p className="muted text-sm doc-breadcrumb">{appTitle} / 使用指南</p>
          <div className="doc-title-row">
            <h1 className="doc-title" id="overview">{t("me.guide.modex_project_guide", { appTitle })}</h1>
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
