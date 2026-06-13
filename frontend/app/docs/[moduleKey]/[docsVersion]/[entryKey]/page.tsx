import { GitBranch } from "lucide-react";
import { getEntries, getModule, getNav, getPage } from "@/lib/api";
import { PageViewTracker } from "@/components/page-view-tracker";
import { DocScope } from "@/components/doc-scope";
import { CodeBlockCopy } from "@/components/code-block-copy";
import { DocReadStats } from "@/components/doc-read-stats";
import { MdxContent } from "@/components/mdx/mdx-content";

export default async function DocPage({ params }: { params: Promise<{ moduleKey: string; docsVersion: string; entryKey: string }> }) {
  const { moduleKey, docsVersion, entryKey } = await params;
  const [module, entries, nav, pagePayload] = await Promise.all([
    getModule(moduleKey),
    getEntries(moduleKey, docsVersion),
    getNav(moduleKey, docsVersion, entryKey),
    getPage(moduleKey, docsVersion, entryKey)
  ]);
  const page = pagePayload.page ?? pagePayload;
  const contentHTML = extractDocumentBody(pagePayload.content_html || "");
  const contentMD = (page.content_md || pagePayload.content_md || "").trim();
  const isMarkdown = page.entry_type === "markdown";

  // VitePress/VuePress/Fumadocs entries ship a full static site (with their own
  // sidebar, search, theme, mermaid/plantuml). Serve it as-is in an iframe from
  // the backend's site route rather than scraping its body into Modex's layout.
  // The site is built with base=/api/docs/{module}/{version}/{entry}/site/, so
  // its absolute asset paths resolve against the backend origin the iframe loads
  // from.
  if (!isMarkdown) {
    const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8671";
    const siteSrc = `${apiBase}/api/docs/${moduleKey}/${docsVersion}/${entryKey}/site/`;
    return (
      <main className="main docs-shell docs-shell--embedded">
        <PageViewTracker docId={page.doc_id} />
        <DocScope moduleKey={module.module_key} moduleName={module.name} />
        <iframe
          className="doc-embed-frame"
          src={siteSrc}
          title={page.title || module.name}
        />
      </main>
    );
  }

  return (
    <main className="main docs-shell">
      <PageViewTracker docId={page.doc_id} />
      <DocScope moduleKey={module.module_key} moduleName={module.name} />
      <section className="doc-layout">
        <aside className="doc-sidebar">
          <div className="doc-nav-head">
            <a className="doc-nav-back" href="/">{module.name}</a>
            <span className="doc-nav-version">{docsVersion}</span>
          </div>
          <nav className="doc-nav">
            {entries.map((entry) => {
              const active = entry.entry_key === entryKey;
              return (
                <a
                  className={`doc-nav-link${active ? " active" : ""}`}
                  key={entry.entry_key}
                  href={`/docs/${moduleKey}/${docsVersion}/${entry.entry_key}`}
                >
                  {entry.title}
                </a>
              );
            })}
          </nav>
        </aside>

        <article className="panel prose doc-page">
          <p className="muted text-sm doc-breadcrumb">{module.category_path}</p>
          <div className="doc-title-row">
            <h1 className="doc-title" id="overview">{page.title}</h1>
            <DocReadStats docId={page.doc_id} />
          </div>
          {page.description ? <p className="doc-lead">{page.description}</p> : null}
          {contentMD ? (
            <MdxContent source={contentMD} />
          ) : contentHTML ? (
            <>
              <div className="embedded-doc-html" dangerouslySetInnerHTML={{ __html: contentHTML }} />
              <CodeBlockCopy />
            </>
          ) : (
            <p>{page.content_text}</p>
          )}
        </article>

        <aside className="doc-toc">
          <div className="doc-toc-card">
            {isMarkdown && nav.length > 0 ? (
              <>
                <p className="doc-toc-title">目录</p>
                <nav className="doc-toc-tree">
                  {nav.map((item) => (
                    <a key={item.path} href={`/docs/${moduleKey}/${docsVersion}/${entryKey}${item.path}`} className="doc-toc-link">
                      {item.title}
                    </a>
                  ))}
                </nav>
              </>
            ) : null}
            <p className={`doc-toc-title${isMarkdown && nav.length > 0 ? " mt-5" : ""}`}>元数据</p>
            <dl className="doc-meta">
              <div><dt>doc_id</dt><dd className="break-all font-mono text-xs">{page.doc_id}</dd></div>
              <div><dt>owner</dt><dd>{page.owner_group || "—"}</dd></div>
              <div><dt>工程版本</dt><dd>{page.package_version || "—"}</dd></div>
            </dl>
            {module.repo_url ? (
              <a className="button doc-toc-source" href={module.repo_url}>
                <GitBranch size={15} />查看源码
              </a>
            ) : null}
          </div>
        </aside>
      </section>
    </main>
  );
}

function extractDocumentBody(html: string) {
  if (!html.trim()) {
    return "";
  }
  const withoutScripts = html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "")
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, "");
  const main = withoutScripts.match(/<main\b[^>]*>([\s\S]*?)<\/main>/i);
  if (main?.[1]) {
    return main[1];
  }
  const body = withoutScripts.match(/<body\b[^>]*>([\s\S]*?)<\/body>/i);
  return body?.[1] || withoutScripts;
}
