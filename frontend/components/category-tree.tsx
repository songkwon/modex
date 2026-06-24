"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import type { Category } from "@/types/modex";
import { categoryRouteSegment } from "@/lib/category-url";
import { useI18n } from "@/lib/i18n";

export function CategoryTree({
  categories,
  activeID,
  rootId,
  backHref,
  backLabel,
  hrefPrefix,
  onSelect
}: {
  categories: Category[];
  activeID?: string;
  rootId?: string;
  backHref?: string;
  backLabel?: string;
  hrefPrefix?: string;
  onSelect?: (category: Category | null) => void;
}) {
  const { t } = useI18n();
  const display = rootId ? findSubtree(categories, rootId) : categories;
  return (
    <aside className="cat-rail rail">
      {backHref ? (
        <Link href={backHref} className="button mb-3 w-full justify-start">
          <ArrowLeft size={15} /> {backLabel || t("component.categoryTree.back")}
        </Link>
      ) : null}
      <div className="cat-rail-head">
        <span className="cat-rail-title">{t("component.categoryTree.documentation_category")}</span>
        {!backHref && activeID ? <button className="quiet-link" onClick={() => onSelect?.(null)}>{t("component.categoryTree.clear")}</button> : null}
      </div>
      <nav className="cat-tree">
        {display.map((category) => (
          <CategoryNode activeID={activeID} category={category} depth={0} key={category.id} hrefPrefix={hrefPrefix} onSelect={onSelect} />
        ))}
      </nav>
    </aside>
  );
}

function findSubtree(categories: Category[], id: string): Category[] {
  for (const cat of categories) {
    if (cat.id === id) {
      return [cat];
    }
    const found = findSubtree(cat.children || [], id);
    if (found.length) return found;
  }
  return [];
}

function CategoryNode({
  category,
  activeID,
  depth,
  hrefPrefix,
  onSelect
}: {
  category: Category;
  activeID?: string;
  depth: number;
  hrefPrefix?: string;
  onSelect?: (category: Category) => void;
}) {
  const hasChildren = !!category.children?.length;
  const href = hrefPrefix ? `${hrefPrefix}/${categoryRouteSegment(category)}` : undefined;
  const className = `cat-node${activeID === category.id ? " active" : ""}${depth === 0 ? " cat-node-root" : ""}`;
  const style = { paddingLeft: 10 + depth * 14 };
  const label = (
    <>
      <span className="cat-node-label">{category.name}</span>
      {hasChildren ? <span className="cat-node-count">{category.children!.length}</span> : null}
    </>
  );
  return (
    <>
      {href ? (
        <Link href={href} className={className} style={style}>
          {label}
        </Link>
      ) : (
        <button className={className} style={style} onClick={() => onSelect?.(category)}>
          {label}
        </button>
      )}
      {hasChildren
        ? category.children!.map((child) => (
            <CategoryNode activeID={activeID} category={child} depth={depth + 1} key={child.id} hrefPrefix={hrefPrefix} onSelect={onSelect} />
          ))
        : null}
    </>
  );
}
