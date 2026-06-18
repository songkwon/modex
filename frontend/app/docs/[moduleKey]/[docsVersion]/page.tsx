import Link from "next/link";
import { getEntries, getModule, getModuleVersions } from "@/lib/api";
import { VersionSwitcher } from "@/components/version-switcher";

export default async function VersionPage({ params }: { params: Promise<{ moduleKey: string; docsVersion: string }> }) {
  const { moduleKey, docsVersion } = await params;
  const [module, entries, versions] = await Promise.all([
    getModule(moduleKey),
    getEntries(moduleKey, docsVersion),
    getModuleVersions(moduleKey).catch(() => [])
  ]);
  return (
    <main className="main">
      <section className="hero-panel">
        <div className="doc-version-hero-row">
          <span className="hero-eyebrow">{module.category_path}</span>
          <VersionSwitcher moduleKey={moduleKey} current={docsVersion} entryKey={entries[0]?.entry_key || ""} versions={versions} />
        </div>
        <h1 className="hero-title">{module.name}</h1>
        <p className="hero-copy">{module.description}</p>
      </section>
      <section className="mt-5 card-grid">
        {entries.length === 0 ? (
          <div className="panel muted">当前版本暂无文档入口。</div>
        ) : entries.map((entry) => (
          <Link className="card" href={`/docs/${moduleKey}/${docsVersion}/${entry.entry_key}`} key={entry.entry_key}>
            <h2 className="text-lg font-semibold">{entry.title}</h2>
            <p className="muted mt-2 text-sm">{entry.entry_type} / {entry.source}</p>
          </Link>
        ))}
      </section>
    </main>
  );
}
