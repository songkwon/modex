import { getCategories } from "@/lib/api";

export default async function AdminCategoriesPage() {
  const categories = await getCategories();
  return (
    <main className="main">
      <section className="panel">
        <h1 className="text-2xl font-semibold">分类管理</h1>
        <pre className="mt-4 overflow-auto text-sm">{JSON.stringify(categories, null, 2)}</pre>
      </section>
    </main>
  );
}
