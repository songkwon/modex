import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { ArrowUpRight, BookOpen } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { getCategories, getEntries, getModules } from "@/lib/api";
import { getServerI18n } from "@/lib/i18n-server";
import type { Category, ModuleInfo } from "@/types/modex";

export default async function CategoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: rawID } = await params;
  const id = decodeURIComponent(rawID);
  const { t } = await getServerI18n();
  const categories = await getCategories();
  const category = findCategory(categories, id);
  if (!category) notFound();

  // A category with no direct modules (e.g. a parent that only holds
  // sub-categories) returns null from the API — coalesce so .length is safe.
  const modules = (await getModules(`?category_id=${encodeURIComponent(id)}`)) ?? [];

  return (
    <main className="main">
      <section className="grid category-grid">
          <CategoryTree
            categories={categories}
            activeID={id}
          rootId={id}
          backHref="/"
          backLabel={t("categories.id.all_categories")}
          hrefPrefix="/categories"
        />
        <div className="category-content">
          <div className="shelf-toolbar mb-5">
            <div>
              <h1>{category.name}</h1>
              <p className="muted">{modules.length} {t("categories.id.document_collections")} {category.description || ""}</p>
            </div>
          </div>
          {modules.length === 1 ? (
            <SingleModuleView module={modules[0]} />
          ) : (
            <div className="package-list">
              {modules.map((module) => (
                <ModuleRow module={module} key={module.module_key} />
              ))}
              {modules.length === 0 ? <div className="empty-state">{t("categories.id.no_document_collections_in_this_category_yet")}</div> : null}
            </div>
          )}
        </div>
      </section>
    </main>
  );
}

// A category with a single doc set opens that doc directly in the full doc
// viewer (which renders Markdown with Modex's own layout, or a site-builder doc
// full-width). Embedding it in an iframe here nested a second Modex chrome
// inside the page ("文档套文档"), so we redirect instead.
async function SingleModuleView({ module }: { module: ModuleInfo }) {
  const { t } = await getServerI18n();
  const entries = await getEntries(module.module_key, module.default_version);
  const primary = entries.find((e) => e.is_primary) || entries[0];
  if (!primary) {
    return <div className="empty-state">{t("categories.id.no_document_entry_points")}</div>;
  }
  redirect(`/docs/${module.module_key}/${module.default_version}/${primary.entry_key}`);
}

async function ModuleRow({ module }: { module: ModuleInfo }) {
  const { t } = await getServerI18n();
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
          <span>{t("categories.id.owner")}: {module.owner_group}</span>
          <span>{t("categories.id.channel")}: {module.channel}</span>
        </div>
      </div>
      <aside className="package-stats text-sm">
        <span className="score-pill">{t("categories.id.activePackage")}</span>
        <div>
          <dt className="muted">{t("categories.id.default_version")}</dt>
          <dd>{module.default_version}</dd>
        </div>
        <div>
          <dt className="muted">{t("categories.id.documentation_type")}</dt>
          <dd className="flex items-center gap-1"><BookOpen size={14} /> {docTypeLabel(module.doc_type)}</dd>
        </div>
        <div>
          <dt className="muted">{t("categories.id.reads_30d")}</dt>
          <dd>{module.reads_30d}</dd>
        </div>
        <div className="flex gap-2 pt-1">
          <Link className="button" href={`/docs/${module.module_key}/${module.default_version}`}>{t("categories.id.open")} <ArrowUpRight size={14} /></Link>
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
