import Link from "next/link";
import { getEntriesSafe, getModule, getModuleVersions } from "@/lib/api";
import { VersionSwitcher } from "@/components/version-switcher";
import { getServerI18n } from "@/lib/i18n-server";

export default async function VersionPage({ params }: { params: Promise<{ moduleKey: string; docsVersion: string }> }) {
  const { moduleKey, docsVersion } = await params;
  const { t } = await getServerI18n();
  const [module, entries, versions] = await Promise.all([
    getModule(moduleKey),
    getEntriesSafe(moduleKey, docsVersion),
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
          <div className="panel muted">{t("docs.module.docsVersion.no_document_entry_is_available_in_the_current")}</div>
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
