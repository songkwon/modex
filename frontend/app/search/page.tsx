"use client";

import { useState } from "react";
import Link from "next/link";
import { Search } from "lucide-react";
import { searchDocs } from "@/lib/api";
import type { SearchResponse } from "@/types/modex";

const modes = ["keyword", "semantic", "hybrid"];

export default function SearchPage() {
  const [query, setQuery] = useState("构建缓存怎么清理");
  const [mode, setMode] = useState("hybrid");
  const [response, setResponse] = useState<SearchResponse | null>(null);

  async function submit() {
    setResponse(await searchDocs({ query, mode, page: 1, page_size: 20, filters: {} }));
  }

  return (
    <main className="main">
      <div className="panel">
        <div className="grid gap-3 md:grid-cols-[1fr_220px_auto]">
          <input value={query} onChange={(e) => setQuery(e.target.value)} />
          <select value={mode} onChange={(e) => setMode(e.target.value)}>
            {modes.map((m) => <option key={m}>{m}</option>)}
          </select>
          <button className="button" onClick={submit}>
            <Search size={16} />
            搜索
          </button>
        </div>
      </div>
      <section className="mt-5 grid gap-4">
        {response ? <p className="muted">共 {response.total} 条结果，模式 {response.mode}</p> : null}
        {response?.results.map((item) => (
          <article className="card" key={item.doc_id}>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <Link href={`/docs/${item.module_key}/${item.docs_version}/${item.path.replace("/", "")}`} className="text-lg font-semibold">{item.title}</Link>
              <span className="tag">{item.score.toFixed(3)}</span>
            </div>
            <p className="muted mt-2 text-sm">{item.module_name} / {item.docs_version} / {item.entry_type} / {item.owner_group}</p>
            <p className="mt-3 leading-7">{item.snippet}</p>
            <div className="mt-3 flex flex-wrap gap-2">
              {item.keywords.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
            </div>
          </article>
        ))}
      </section>
    </main>
  );
}
