import { GitBranch, Search } from "lucide-react";
import { getEntries, getModule, getPage } from "@/lib/api";

export default async function DocPage({ params }: { params: { moduleKey: string; docsVersion: string; entryKey: string } }) {
  const [module, entries, page] = await Promise.all([
    getModule(params.moduleKey),
    getEntries(params.moduleKey, params.docsVersion),
    getPage(params.moduleKey, params.docsVersion, params.entryKey)
  ]);
  return (
    <main className="main">
      <div className="panel mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="muted text-sm">{module.category_path} / {params.docsVersion}</p>
          <h1 className="text-xl font-semibold">{page.title}</h1>
        </div>
        <div className="flex gap-2">
          <button className="button"><Search size={16} />本模块搜索</button>
          <a className="button" href={module.repo_url}><GitBranch size={16} />查看源码</a>
        </div>
      </div>
      <section className="doc-layout">
        <aside className="panel">
          <h2 className="font-semibold">Entry</h2>
          <div className="mt-3 grid gap-2 text-sm">
            {entries.map((entry) => <a key={entry.entry_key} href={`/docs/${params.moduleKey}/${params.docsVersion}/${entry.entry_key}`}>{entry.title}</a>)}
          </div>
        </aside>
        <article className="panel prose">
          <h2 id="overview">{page.title}</h2>
          <p>{page.description}</p>
          <h2 id="content">正文</h2>
          <p>{page.content_text}</p>
        </article>
        <aside className="panel">
          <h2 className="font-semibold">本文目录</h2>
          <div className="mt-3 grid gap-2 text-sm">
            <a href="#overview">概览</a>
            <a href="#content">正文</a>
            <a href="#metadata">元数据</a>
          </div>
          <h2 className="mt-5 font-semibold" id="metadata">元数据</h2>
          <dl className="mt-3 grid gap-2 text-sm">
            <div><dt className="muted">doc_id</dt><dd>{page.doc_id}</dd></div>
            <div><dt className="muted">owner</dt><dd>{page.owner_group}</dd></div>
            <div><dt className="muted">工程版本</dt><dd>{page.package_version}</dd></div>
          </dl>
        </aside>
      </section>
    </main>
  );
}
