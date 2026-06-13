import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import type { Category } from "@/types/modex";

export function CategoryTree({
  categories,
  activeID,
  rootId,
  backHref,
  backLabel,
  hrefFor,
  onSelect
}: {
  categories: Category[];
  activeID?: string;
  rootId?: string;
  backHref?: string;
  backLabel?: string;
  hrefFor?: (category: Category) => string;
  onSelect?: (category: Category | null) => void;
}) {
  const display = rootId ? findSubtree(categories, rootId) : categories;
  return (
    <aside className="cat-rail rail">
      {backHref ? (
        <Link href={backHref} className="button mb-3 w-full justify-start">
          <ArrowLeft size={15} /> {backLabel || "返回"}
        </Link>
      ) : null}
      <div className="cat-rail-head">
        <span className="cat-rail-title">文档分类</span>
        {!backHref && activeID ? <button className="quiet-link" onClick={() => onSelect?.(null)}>清除</button> : null}
      </div>
      <nav className="cat-tree">
        {display.map((category) => (
          <CategoryNode activeID={activeID} category={category} depth={0} key={category.id} hrefFor={hrefFor} onSelect={onSelect} />
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
  hrefFor,
  onSelect
}: {
  category: Category;
  activeID?: string;
  depth: number;
  hrefFor?: (category: Category) => string;
  onSelect?: (category: Category) => void;
}) {
  const hasChildren = !!category.children?.length;
  const href = hrefFor?.(category);
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
            <CategoryNode activeID={activeID} category={child} depth={depth + 1} key={child.id} hrefFor={hrefFor} onSelect={onSelect} />
          ))
        : null}
    </>
  );
}
