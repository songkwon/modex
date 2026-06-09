import Link from "next/link";
import { getModules } from "@/lib/api";

export default async function AdminModulesPage() {
  const modules = await getModules();
  return (
    <main className="main">
      <section className="panel">
        <h1 className="text-2xl font-semibold">模块管理</h1>
      </section>
      <section className="mt-5 grid gap-3">
        {modules.map((m) => (
          <Link className="card" href={`/admin/modules/${m.module_key}`} key={m.module_key}>
            <h2 className="font-semibold">{m.name}</h2>
            <p className="muted mt-1 text-sm">{m.owner_group} / {m.package_version}</p>
          </Link>
        ))}
      </section>
    </main>
  );
}
