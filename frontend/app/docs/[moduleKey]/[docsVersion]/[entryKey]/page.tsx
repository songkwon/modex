import { GitBranch } from "lucide-react";
import { getEntries, getModule, getPage } from "@/lib/api";
import { PageViewTracker } from "@/components/page-view-tracker";
import { DocScope } from "@/components/doc-scope";

export default async function DocPage({ params }: { params: { moduleKey: string; docsVersion: string; entryKey: string } }) {
  const [module, entries, pagePayload] = await Promise.all([
    getModule(params.moduleKey),
    getEntries(params.moduleKey, params.docsVersion),
    getPage(params.moduleKey, params.docsVersion, params.entryKey)
  ]);
  const page = pagePayload.page ?? pagePayload;
  const contentHTML = extractDocumentBody(pagePayload.content_html || "");
  return (
    <main className="main docs-shell">
      <PageViewTracker docId={page.doc_id} />
      <DocScope moduleKey={module.module_key} moduleName={module.name} />
      <div className="panel mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="muted text-sm">{module.category_path} / {params.docsVersion}</p>
          <h1 className="text-xl font-semibold">{module.name}</h1>
        </div>
        <div className="flex gap-2">
          <a className="button button-primary" href={module.repo_url}><GitBranch size={16} />查看源码</a>
        </div>
      </div>
      <section className="doc-layout">
        <aside className="panel doc-sidebar">
          <div className="section-heading">
            <h2>Entries</h2>
            <span className="tag">{params.docsVersion}</span>
          </div>
          <div className="mt-3 grid gap-1 text-sm">
            {entries.map((entry) => (
              <a className="category-node" key={entry.entry_key} href={`/docs/${params.moduleKey}/${params.docsVersion}/${entry.entry_key}`}>
                <span>{entry.title}</span>
              </a>
            ))}
          </div>
        </aside>
        <article className="panel prose doc-page">
          <p className="hero-eyebrow">Documentation</p>
          <h1 className="doc-title" id="overview">{page.title}</h1>
          <p className="hero-copy">{page.description}</p>
          <h2 id="content">正文</h2>
          {contentHTML ? (
            <div className="embedded-doc-html" dangerouslySetInnerHTML={{ __html: contentHTML }} />
          ) : (
            <p>{page.content_text}</p>
          )}
          <h2>AI 访问</h2>
          <p>该页面可以通过 MCP 的 get_doc_page 工具读取，也可以通过 documents.jsonl 做精细召回。</p>
        </article>
        <aside className="panel doc-toc">
          <div className="section-heading">
            <h2>On this page</h2>
          </div>
          <div className="mt-3 grid gap-2 text-sm">
            <a className="category-node" href="#overview">概览</a>
            <a className="category-node" href="#content">正文</a>
            <a className="category-node" href="#metadata">元数据</a>
          </div>
          <h2 className="mt-5 font-semibold" id="metadata">元数据</h2>
          <dl className="mt-3 grid gap-3 text-sm">
            <div><dt className="muted">doc_id</dt><dd className="break-all font-mono text-xs">{page.doc_id}</dd></div>
            <div><dt className="muted">owner</dt><dd>{page.owner_group}</dd></div>
            <div><dt className="muted">工程版本</dt><dd>{page.package_version}</dd></div>
          </dl>
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
