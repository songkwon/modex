import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { ArrowUpRight, BookOpen, FolderTree } from "lucide-react";
import { CategoryTree } from "@/components/category-tree";
import { CategoryInfoButton } from "@/components/category-info-button";
import { getCategories, getEntriesSafe, getModules } from "@/lib/api";
import { categoryHref, findCategoryByRouteSegment } from "@/lib/category-url";
import { getServerI18n } from "@/lib/i18n-server";
import type { Category, ModuleInfo } from "@/types/modex";

export default async function CategoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: segment } = await params;
  const { t } = await getServerI18n();
  const categories = await getCategories();
  const category = findCategoryByRouteSegment(categories, segment);
  if (!category) notFound();
  const id = category.id;

  const allModules = (await getModules().catch(() => [])) ?? [];
  const childCategories = category.children || [];
  const categoryIDSet = new Set(categoryIDs(category));
  const modules = allModules.filter((m) => (m.category_ids || []).some((categoryID) => categoryIDSet.has(categoryID)));

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
              <div className="category-title-row">
                <h1>{category.name}</h1>
                <CategoryInfoButton category={category} modulesCount={modules.length} />
              </div>
              <p className="muted">{modules.length} {t("categories.id.document_collections")} {category.description || ""}</p>
              {category.responsible_team_info ? (
                <p className="category-owner-summary">
                  {t("component.categoryInfo.responsible_team")}: {category.responsible_team_info.name || category.responsible_team_info.key}
                  {(category.responsible_team_info.leaders || []).length > 0 ? ` · ${t("component.categoryInfo.owners")}: ${category.responsible_team_info.leaders.join(", ")}` : ""}
                </p>
              ) : null}
            </div>
          </div>
          {childCategories.length > 0 ? (
            <SubcategoryCards categories={childCategories} modules={allModules} />
          ) : null}
          {modules.length === 1 && childCategories.length === 0 ? (
            <SingleModuleView module={modules[0]} />
          ) : (
            <>
              <div className="shelf-toolbar document-list-heading mb-3">
                <div>
                  <h2>{t("categories.id.document_list")}</h2>
                  <p>{t("categories.id.document_list_hint")}</p>
                </div>
              </div>
              <div className="package-list">
                {modules.map((module) => (
                  <ModuleRow module={module} key={module.module_key} />
                ))}
                {modules.length === 0 ? <div className="empty-state">{t("categories.id.no_document_collections_in_this_category_yet")}</div> : null}
              </div>
            </>
          )}
        </div>
      </section>
    </main>
  );
}

async function SubcategoryCards({ categories, modules }: { categories: Category[]; modules: ModuleInfo[] }) {
  const { t } = await getServerI18n();
  return (
    <section className="subcategory-section">
      <div className="shelf-toolbar mb-3">
        <div>
          <h2>{t("categories.id.subcategories")}</h2>
          <p>{t("categories.id.subcategories_hint")}</p>
        </div>
      </div>
      <div className="subcategory-grid">
        {categories.map((cat) => {
          const total = countModules(modules, cat);
          const childCount = cat.children?.length || 0;
          return (
            <Link className="subcategory-card" href={categoryHref(cat)} key={cat.id}>
              <span className="subcategory-icon"><FolderTree size={18} /></span>
              <span className="subcategory-body">
                <span className="subcategory-name">{cat.name}</span>
                {cat.description ? <span className="subcategory-desc">{cat.description}</span> : null}
                <span className="subcategory-meta">
                  {total} {t("categories.id.document_collections_short")}
                  {childCount > 0 ? ` · ${childCount} ${t("categories.id.subcategories_short")}` : ""}
                </span>
              </span>
              <ArrowUpRight size={15} className="subcategory-arrow" />
            </Link>
          );
        })}
      </div>
    </section>
  );
}

// A category with a single doc set opens that doc directly in the full doc
// viewer (which renders Markdown with Modex's own layout, or a site-builder doc
// full-width). Embedding it in an iframe here nested a second Modex chrome
// inside the page ("文档套文档"), so we redirect instead.
async function SingleModuleView({ module }: { module: ModuleInfo }) {
  const { t } = await getServerI18n();
  if (!module.default_version) {
    return <div className="empty-state">{t("categories.id.no_document_entry_points")}</div>;
  }
  const entries = await getEntriesSafe(module.module_key, module.default_version);
  const primary = entries.find((e) => e.is_primary) || entries[0];
  if (!primary) {
    return <div className="empty-state">{t("categories.id.no_document_entry_points")}</div>;
  }
  redirect(`/docs/${module.module_key}/${module.default_version}/${primary.entry_key}`);
}

async function ModuleRow({ module }: { module: ModuleInfo }) {
  const { t } = await getServerI18n();
  const hasPublishedDocs = !!module.default_version;
  const entries = hasPublishedDocs ? await getEntriesSafe(module.module_key, module.default_version) : [];
  const primary = entries.find((e) => e.is_primary) || entries[0];
  const href = primary ? `/docs/${module.module_key}/${module.default_version}/${primary.entry_key}` : "";
  const body = (
    <>
      <div>
        <div className="package-name">
          <span className="status-dot" />
          <span>{module.name}</span>
        </div>
        <div className="package-meta">
          {module.category_path ? <span>{module.category_path}</span> : null}
          {module.owner_group ? <span>{t("categories.id.owner")}: {module.owner_group}</span> : null}
          {module.channel ? <span>{t("categories.id.channel")}: {module.channel}</span> : null}
        </div>
      </div>
      <aside className="package-stats text-sm">
        <div>
          <dt className="muted">{t("categories.id.default_version")}</dt>
          <dd>{module.default_version || "-"}</dd>
        </div>
        <div>
          <dt className="muted">{t("categories.id.documentation_type")}</dt>
          <dd className="flex items-center gap-1"><BookOpen size={14} /> {docTypeLabel(module.doc_type)}</dd>
        </div>
        <div>
          <dt className="muted">{t("categories.id.reads_30d")}</dt>
          <dd>{module.reads_30d}</dd>
        </div>
        {href ? <ArrowUpRight size={16} className="muted" /> : null}
      </aside>
    </>
  );
  if (href) {
    return <Link className="package-row" href={href}>{body}</Link>;
  }
  return (
    <article className="package-row">
      {body}
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

function categoryIDs(category: Category): string[] {
  return [category.id, ...(category.children || []).flatMap(categoryIDs)];
}

function countModules(modules: ModuleInfo[], category: Category) {
  const ids = new Set(categoryIDs(category));
  return modules.filter((m) => (m.category_ids || []).some((categoryID) => ids.has(categoryID))).length;
}
