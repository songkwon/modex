import type { Category } from "@/types/modex";

export function CategoryTree({
  categories,
  activeID,
  onSelect
}: {
  categories: Category[];
  activeID?: string;
  onSelect?: (category: Category | null) => void;
}) {
  return (
    <div className="panel rail">
      <div className="section-heading">
        <h2>文档分类</h2>
        {activeID ? (
          <button className="quiet-link" onClick={() => onSelect?.(null)}>清除</button>
        ) : (
          <span className="tag">Registry</span>
        )}
      </div>
      <div className="mt-3 grid gap-1">
        {categories.map((category) => (
          <CategoryNode activeID={activeID} category={category} key={category.id} onSelect={onSelect} />
        ))}
      </div>
    </div>
  );
}

function CategoryNode({
  category,
  activeID,
  onSelect
}: {
  category: Category;
  activeID?: string;
  onSelect?: (category: Category) => void;
}) {
  return (
    <div>
      <button className={`category-node w-full ${activeID === category.id ? "active" : ""}`} onClick={() => onSelect?.(category)}>
        <span>{category.name}</span>
        {category.children?.length ? <span className="muted text-xs">{category.children.length}</span> : null}
      </button>
      {category.children && category.children.length > 0 ? (
        <div className="ml-3 border-l border-border pl-2">
          {category.children.map((child) => (
            <CategoryNode activeID={activeID} category={child} key={child.id} onSelect={onSelect} />
          ))}
        </div>
      ) : null}
    </div>
  );
}
