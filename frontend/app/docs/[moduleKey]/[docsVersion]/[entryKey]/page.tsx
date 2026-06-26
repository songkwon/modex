import { GitBranch } from "lucide-react";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { getEntriesSafe, getModule, getModuleVersions, getNav, getPage } from "@/lib/api";
import { PageViewTracker } from "@/components/page-view-tracker";
import { DocScope } from "@/components/doc-scope";
import { CodeBlockCopy } from "@/components/code-block-copy";
import { DocReadStats } from "@/components/doc-read-stats";
import { MdxContent } from "@/components/mdx/mdx-content";
import { DocToc } from "@/components/doc-toc";
import { VersionSwitcher } from "@/components/version-switcher";
import { DocFooter, FloatingDocFeedback } from "@/components/doc-footer";
import { DocSourceToggle } from "@/components/doc-source-toggle";
import { getServerI18n } from "@/lib/i18n-server";
import { publicApiBaseURL } from "@/lib/runtime-config";

export default async function DocPage({
  params,
  searchParams,
}: {
  params: Promise<{ moduleKey: string; docsVersion: string; entryKey: string }>;
  searchParams: Promise<{ p?: string }>;
}) {
  const { moduleKey, docsVersion, entryKey } = await params;
  const { p: deepLink } = await searchParams;
  const { t } = await getServerI18n();
  const [module, entries, versions, navTree, pagePayload] = await Promise.all([
    getModule(moduleKey),
    getEntriesSafe(moduleKey, docsVersion),
    getModuleVersions(moduleKey).catch(() => []),
    getNav(moduleKey, docsVersion, entryKey).catch(() => []),
    getPage(moduleKey, docsVersion, entryKey).catch(() => null)
  ]);
  if (!pagePayload) {
    notFound();
  }
  const page = pagePayload.page ?? pagePayload;
  const contentHTML = extractDocumentBody(pagePayload.content_html || "");
  const contentMD = (page.content_md || pagePayload.content_md || "").trim();
  const isMarkdown = page.entry_type === "markdown";
  const apiBase = publicApiBaseURL();
  const actualEntryKey = page.entry_key || entryKey;
  const assetBase = `${apiBase}/api/docs/${moduleKey}/${docsVersion}/${actualEntryKey}/site/`;
  // Prev/next doc cards from the entry order.
  const idx = entries.findIndex((e) => e.entry_key === actualEntryKey);
  const toLink = (e: any) => (e ? { title: e.title, href: `/docs/${moduleKey}/${docsVersion}/${e.entry_key}` } : undefined);
  const prevDoc = idx > 0 ? toLink(entries[idx - 1]) : undefined;
  const nextDoc = idx >= 0 && idx < entries.length - 1 ? toLink(entries[idx + 1]) : undefined;
  const sourceMap = buildSourceMap(entries);

  // VitePress/VuePress/Fumadocs entries ship a full static site (with their own
  // sidebar, search, theme, mermaid/plantuml). Serve it as-is in an iframe from
  // the backend's site route rather than scraping its body into Modex's layout.
  // The site is built with base=/api/docs/{module}/{version}/{entry}/site/, so
  // its absolute asset paths resolve against the backend origin the iframe loads
  // from.
  if (!isMarkdown) {
    // Deep-link into the built site: a search hit passes ?p=<route> (e.g.
    // /posts/cbb/x). Convert it to the built file path (posts/cbb/x.html;
    // dir routes keep their trailing slash to load that dir's index.html).
    let rel = (deepLink || "").replace(/^\/+/, "");
    if (rel && !rel.endsWith("/")) rel += ".html";
    const siteSrc = `${apiBase}/api/docs/${moduleKey}/${docsVersion}/${actualEntryKey}/site/${rel}`;
    return (
      <main className="doc-fullscreen">
        <PageViewTracker docId={page.doc_id} title={page.title || module.name} moduleKey={module.module_key} moduleName={module.name} docsVersion={docsVersion} entryKey={actualEntryKey} />
        <DocScope moduleKey={module.module_key} moduleName={module.name} />
        <div className="doc-version-dock">
          <VersionSwitcher moduleKey={moduleKey} current={docsVersion} entryKey={actualEntryKey} versions={versions} />
        </div>
        <FloatingDocFeedback docId={page.doc_id} />
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
      <PageViewTracker docId={page.doc_id} title={page.title || module.name} moduleKey={module.module_key} moduleName={module.name} docsVersion={docsVersion} entryKey={actualEntryKey} />
      <DocScope moduleKey={module.module_key} moduleName={module.name} />
      <section className="doc-layout">
        <aside className="doc-sidebar">
          <div className="doc-nav-head">
            <a className="doc-nav-back" href="/">{module.name}</a>
            <VersionSwitcher moduleKey={moduleKey} current={docsVersion} entryKey={actualEntryKey} versions={versions} />
          </div>
          <nav className="doc-nav">
            {navTree.length > 0 ? renderDocNavTree(navTree, moduleKey, docsVersion, actualEntryKey) : entries.map((entry) => {
              const active = entry.entry_key === actualEntryKey;
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
          {contentMD ? (
            <DocSourceToggle source={contentMD}>
              <MdxContent
                source={contentMD}
                assetBase={assetBase}
                docLinks={{ moduleKey, docsVersion, sourceFile: page.source_file, sourceMap }}
              />
            </DocSourceToggle>
          ) : contentHTML ? (
            <>
              <div className="embedded-doc-html" dangerouslySetInnerHTML={{ __html: contentHTML }} />
              <CodeBlockCopy />
            </>
          ) : (
            <p>{page.content_text}</p>
          )}
          <DocFooter docId={page.doc_id} prev={prevDoc} next={nextDoc} />
        </article>

        <aside className="doc-toc">
          <div className="doc-toc-card">
            {isMarkdown ? <DocToc /> : null}
            {module.repo_url ? (
              <a className="button doc-toc-source" href={module.repo_url}>
                <GitBranch size={15} />{t("component.moduleDrawer.view_source_code")}
              </a>
            ) : null}
          </div>
        </aside>
      </section>
    </main>
  );
}

function buildSourceMap(entries: any[]) {
  const out: Record<string, string> = {};
  for (const entry of entries || []) {
    const source = normalizeSourcePath(entry?.source || "");
    const key = String(entry?.entry_key || "");
    if (!source || !key) continue;
    out[source] = key;
    out[source.toLowerCase()] = key;
  }
  return out;
}

function normalizeSourcePath(source: string) {
  const parts: string[] = [];
  for (const part of source.replace(/\\/g, "/").split("/")) {
    if (!part || part === ".") continue;
    if (part === "..") {
      parts.pop();
      continue;
    }
    parts.push(part);
  }
  return parts.join("/");
}

type DocNavItem = { title: string; path: string; children?: DocNavItem[] };

function renderDocNavTree(items: DocNavItem[], moduleKey: string, docsVersion: string, activeEntryKey: string, depth = 0): ReactNode {
  return items.map((item, idx) => {
    const entryKey = item.path?.startsWith("/") && !item.path.startsWith("#") ? item.path.slice(1) : "";
    const active = entryKey === activeEntryKey;
    if (!entryKey) {
      return (
        <div className="doc-nav-group" key={`${item.title}-${idx}`}>
          <div className="doc-nav-folder" style={{ paddingLeft: 10 + depth * 12 }}>{item.title}</div>
          {item.children?.length ? renderDocNavTree(item.children, moduleKey, docsVersion, activeEntryKey, depth + 1) : null}
        </div>
      );
    }
    return (
      <a
        className={`doc-nav-link${active ? " active" : ""}`}
        key={`${entryKey}-${idx}`}
        href={`/docs/${moduleKey}/${docsVersion}/${entryKey}`}
        style={{ paddingLeft: 10 + depth * 12 }}
      >
        {item.title}
      </a>
    );
  });
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
