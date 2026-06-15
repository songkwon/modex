import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { ArrowUpRight, BookOpen } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { getCategories, getEntries, getModules } from "@/lib/api";
import type { Category, ModuleInfo } from "@/types/modex";

export default async function CategoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const categories = await getCategories();
  const category = findCategory(categories, id);
  if (!category) notFound();

  const modules = await getModules(`?category_id=${encodeURIComponent(id)}`);

  return (
    <main className="main">
      <section className="grid category-grid">
          <CategoryTree
            categories={categories}
            activeID={id}
            rootId={id}
          backHref="/"
          backLabel="全部分类"
          hrefFor={(c) => `/categories/${c.id}`}
        />
        <div className="category-content">
          <div className="shelf-toolbar mb-5">
            <div>
              <h1>{category.name}</h1>
              <p className="muted">{modules.length} 个文档集合 · {category.description || ""}</p>
            </div>
          </div>
          {modules.length === 1 ? (
            <SingleModuleView module={modules[0]} />
          ) : (
            <div className="package-list">
              {modules.map((module) => (
                <ModuleRow module={module} key={module.module_key} />
              ))}
              {modules.length === 0 ? <div className="empty-state">该分类下暂无文档集合</div> : null}
            </div>
          )}
        </div>
      </section>
    </main>
  );
}

async function SingleModuleView({ module }: { module: ModuleInfo }) {
  const isStaticSite = module.doc_type === "vitepress" || module.doc_type === "vuepress" || module.doc_type === "fumadocs";
  if (isStaticSite) {
    // Site-builder docs ship their own full chrome, so open them in the
    // full-width, no-Modex-nav viewer rather than a cramped embedded frame.
    const entries = await getEntries(module.module_key, module.default_version);
    const primary = entries.find((e) => e.is_primary) || entries[0];
    if (!primary) {
      return <div className="empty-state">暂无文档入口</div>;
    }
    redirect(`/docs/${module.module_key}/${module.default_version}/${primary.entry_key}`);
  }
  return <MarkdownModuleView module={module} />;
}

async function MarkdownModuleView({ module }: { module: ModuleInfo }) {
  const entries = await getEntries(module.module_key, module.default_version);
  const primary = entries.find((e) => e.is_primary) || entries[0];
  if (!primary) {
    return <div className="empty-state">暂无文档入口</div>;
  }
  return (
    <div className="embedded-doc-frame">
      <Link href={`/docs/${module.module_key}/${module.default_version}/${primary.entry_key}`} className="button mb-3 inline-flex">
        在文档页中打开 <ArrowUpRight size={14} />
      </Link>
      <iframe
        src={`/docs/${module.module_key}/${module.default_version}/${primary.entry_key}`}
        title={module.name}
        className="doc-iframe"
        sandbox="allow-same-origin allow-scripts allow-popups"
      />
    </div>
  );
}

function ModuleRow({ module }: { module: ModuleInfo }) {
  return (
    <article className="package-row">
      <div>
        <Link href={`/docs/${module.module_key}/${module.default_version}`} className="min-w-0">
          <div className="package-name">
            <span className="status-dot" />
            <span>{module.name}</span>
          </div>
        </Link>
        <p className="muted mt-3 text-sm leading-6">{module.description}</p>
        <div className="package-meta">
          <span>{module.category_path}</span>
          <span>owner: {module.owner_group}</span>
          <span>channel: {module.channel}</span>
        </div>
      </div>
      <aside className="package-stats text-sm">
        <span className="score-pill">active package</span>
        <div>
          <dt className="muted">默认版本</dt>
          <dd>{module.default_version}</dd>
        </div>
        <div>
          <dt className="muted">文档类型</dt>
          <dd className="flex items-center gap-1"><BookOpen size={14} /> {docTypeLabel(module.doc_type)}</dd>
        </div>
        <div>
          <dt className="muted">阅读 30d</dt>
          <dd>{module.reads_30d}</dd>
        </div>
        <div className="flex gap-2 pt-1">
          <Link className="button" href={`/docs/${module.module_key}/${module.default_version}`}>打开 <ArrowUpRight size={14} /></Link>
        </div>
      </aside>
    </article>
  );
}

function docTypeLabel(t?: string) {
  switch (t) {
    case "vitepress": return "VitePress";
    case "vuepress": return "VuePress";
    case "fumadocs": return "Fumadocs";
    default: return "Markdown";
  }
}

function findCategory(categories: Category[], id: string): Category | null {
  for (const cat of categories) {
    if (cat.id === id) return cat;
    const found = findCategory(cat.children || [], id);
    if (found) return found;
  }
  return null;
}
