import { getModule } from "@/lib/api";

export default async function AdminModuleDetail({ params }: { params: { moduleKey: string } }) {
  const module = await getModule(params.moduleKey);
  return (
    <main className="main">
      <section className="panel">
        <h1 className="text-2xl font-semibold">{module.name}</h1>
        <pre className="mt-4 overflow-auto text-sm">{JSON.stringify(module, null, 2)}</pre>
      </section>
    </main>
  );
}
