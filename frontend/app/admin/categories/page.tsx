import { AdminShell } from "@/components/admin-shell";
import { getCategories } from "@/lib/api";
import type { Category } from "@/types/modex";

export default async function AdminCategoriesPage() {
  const categories = await getCategories();
  const rows = flatten(categories);
  return (
    <AdminShell title="分类管理" kicker="Registry" description="分类由平台 Registry 统一治理，项目仓库的 docs.yaml 不直接决定平台分类。">
      <section className="panel">
        <table className="data-table">
          <thead>
            <tr>
              <th>分类</th>
              <th>Key</th>
              <th>描述</th>
              <th>层级</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.category.id}>
                <td className="font-medium">{"— ".repeat(row.depth)}{row.category.name}</td>
                <td><span className="code-chip">{row.category.key}</span></td>
                <td className="muted">{row.category.description}</td>
                <td>{row.depth + 1}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}

function flatten(categories: Category[], depth = 0): Array<{ category: Category; depth: number }> {
  return categories.flatMap((category) => [
    { category, depth },
    ...flatten(category.children || [], depth + 1)
  ]);
}
