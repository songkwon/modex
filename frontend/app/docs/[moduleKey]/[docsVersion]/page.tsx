import Link from "next/link";
import { getEntries, getModule } from "@/lib/api";

export default async function VersionPage({ params }: { params: { moduleKey: string; docsVersion: string } }) {
  const [module, entries] = await Promise.all([
    getModule(params.moduleKey),
    getEntries(params.moduleKey, params.docsVersion)
  ]);
  return (
    <main className="main">
      <section className="hero-panel">
        <span className="hero-eyebrow">{module.category_path}</span>
        <h1 className="hero-title">{module.name}</h1>
        <p className="hero-copy">{module.description}</p>
      </section>
      <section className="mt-5 card-grid">
        {entries.map((entry) => (
          <Link className="card" href={`/docs/${params.moduleKey}/${params.docsVersion}/${entry.entry_key}`} key={entry.entry_key}>
            <h2 className="text-lg font-semibold">{entry.title}</h2>
            <p className="muted mt-2 text-sm">{entry.entry_type} / {entry.source}</p>
          </Link>
        ))}
      </section>
    </main>
  );
}
