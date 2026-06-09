import { api } from "@/lib/api";

export default async function AnalyticsPage() {
  const data = await api<any>("/api/admin/analytics/pages");
  return (
    <main className="main">
      <section className="panel">
        <h1 className="text-2xl font-semibold">阅读统计</h1>
        <pre className="mt-4 overflow-auto text-sm">{JSON.stringify(data, null, 2)}</pre>
      </section>
    </main>
  );
}
