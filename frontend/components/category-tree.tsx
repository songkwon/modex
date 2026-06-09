import type { Category } from "@/types/modex";

export function CategoryTree({ categories }: { categories: Category[] }) {
  return (
    <div className="panel">
      <h2 className="text-base font-semibold">分类</h2>
      <div className="mt-3 grid gap-2">
        {categories.map((category) => (
          <CategoryNode category={category} key={category.id} />
        ))}
      </div>
    </div>
  );
}

function CategoryNode({ category }: { category: Category }) {
  return (
    <div>
      <div className="rounded-md border border-transparent px-2 py-1.5 text-sm font-medium hover:border-border">
        {category.name}
      </div>
      {category.children && category.children.length > 0 ? (
        <div className="ml-3 border-l border-border pl-2">
          {category.children.map((child) => (
            <CategoryNode category={child} key={child.id} />
          ))}
        </div>
      ) : null}
    </div>
  );
}
