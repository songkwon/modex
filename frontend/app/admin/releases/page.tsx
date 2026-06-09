import { api } from "@/lib/api";

export default async function ReleasesPage() {
  const releases = await api<any[]>("/api/admin/releases");
  return (
    <main className="main">
      <section className="panel">
        <h1 className="text-2xl font-semibold">发布记录</h1>
        <pre className="mt-4 overflow-auto text-sm">{JSON.stringify(releases, null, 2)}</pre>
      </section>
    </main>
  );
}
