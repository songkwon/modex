"use client";

import { useState } from "react";
import Link from "next/link";
import { Filter, Search, X } from "lucide-react";
import { searchDocs } from "@/lib/api";
import { highlight } from "@/lib/highlight";
import type { SearchResponse } from "@/types/modex";

const modes = ["keyword", "semantic", "hybrid"];
type Filters = {
  modules?: string[];
  entry_types?: string[];
  owners?: string[];
  docs_versions?: string[];
  category_ids?: string[];
};

export default function SearchPage() {
  const [query, setQuery] = useState("构建缓存怎么清理");
  const [mode, setMode] = useState("hybrid");
  const [response, setResponse] = useState<SearchResponse | null>(null);
  const [filters, setFilters] = useState<Filters>({});

  async function submit(nextFilters = filters, nextMode = mode) {
    setResponse(await searchDocs({ query, mode: nextMode, page: 1, page_size: 20, filters: nextFilters }));
  }

  async function toggleFilter(key: keyof Filters, value: string) {
    const current = filters[key] || [];
    const nextValues = current.includes(value) ? current.filter((x) => x !== value) : [value];
    const next = { ...filters, [key]: nextValues.length ? nextValues : undefined };
    setFilters(next);
    await submit(next);
  }

  async function clearFilters() {
    setFilters({});
    await submit({});
  }

  async function selectMode(nextMode: string) {
    setMode(nextMode);
    await submit(filters, nextMode);
  }

  return (
    <main className="main">
      <section className="hero-panel mb-5">
        <span className="hero-eyebrow">
          <Search size={15} />
          Search Registry
        </span>
        <h1 className="hero-title">在所有模块文档中检索答案</h1>
        <p className="hero-copy">跨模块搜索文档内容，并按模块、版本、文档类型和 Owner 收窄结果。</p>
        <div className="command-search search-command">
          <Search size={19} className="text-blue-600" />
          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="例如：构建缓存怎么清理" />
          <select value={mode} onChange={(e) => setMode(e.target.value)}>
            {modes.map((m) => <option key={m}>{m}</option>)}
          </select>
          <button className="button button-primary" onClick={() => submit()}>
            <Search size={16} />
            搜索
          </button>
        </div>
      </section>
      <section className="search-layout">
        <div className="grid gap-4">
          <div className="panel">
          <div className="section-heading">
            <h2>搜索结果</h2>
            {response ? <span className="muted text-sm">共 {response.total} 条，模式 {response.mode}</span> : <span className="muted text-sm">等待搜索</span>}
          </div>
            <p className="muted text-sm">结果保留模块、版本、Owner 和 Entry 类型，便于确认来源。</p>
          </div>
          {response?.results.map((item) => (
            <article className="card result-card" key={item.doc_id}>
              <p className="ds-crumb">{item.breadcrumb} · {item.docs_version} · {item.entry_type}</p>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <Link href={item.path} className="text-xl font-semibold">{highlight(item.title, item.match_terms)}</Link>
                <span className="tag">score {item.score.toFixed(3)}</span>
              </div>
              <p className="mt-3 leading-7" style={{ color: "hsl(var(--muted))" }}>{highlight(item.snippet, item.match_terms)}</p>
              <div className="mt-4 flex flex-wrap gap-2">
                {item.keywords.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
              </div>
            </article>
          ))}
          {response && response.results.length === 0 ? <div className="empty-state">没有匹配结果，换一个关键词或清除筛选条件。</div> : null}
        </div>
        <aside className="panel rail">
          <div className="section-heading">
            <h2>筛选面板</h2>
            <Filter size={16} className="text-blue-600" />
          </div>
          <div className="grid gap-4 text-sm">
            <div>
              <p className="font-medium">搜索模式</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {modes.map((m) => <button className={`button ${mode === m ? "button-primary" : ""}`} key={m} onClick={() => selectMode(m)}>{m}</button>)}
              </div>
            </div>
            {Object.values(filters).some(Boolean) ? (
              <button className="button" onClick={clearFilters}><X size={15} />清除筛选</button>
            ) : null}
            {response?.facets ? (
              <div className="grid gap-5">
                <FacetGroup active={filters.category_ids || []} items={response.facets.categories || {}} label="文档分类" onToggle={(value) => toggleFilter("category_ids", value)} />
                <FacetGroup active={filters.entry_types || []} items={response.facets.entry_types || {}} label="文档类型" onToggle={(value) => toggleFilter("entry_types", value)} />
                <FacetGroup active={filters.owners || []} items={response.facets.owners || {}} label="所有者" onToggle={(value) => toggleFilter("owners", value)} />
                <FacetGroup active={filters.modules || []} items={response.facets.modules || {}} label="模块" onToggle={(value) => toggleFilter("modules", value)} />
                <FacetGroup active={filters.docs_versions || []} items={response.facets.docs_versions || {}} label="版本" onToggle={(value) => toggleFilter("docs_versions", value)} />
              </div>
            ) : null}
          </div>
        </aside>
      </section>
    </main>
  );
}

function FacetGroup({
  label,
  items,
  active,
  onToggle
}: {
  label: string;
  items: Record<string, number>;
  active: string[];
  onToggle: (value: string) => void;
}) {
  return (
    <div>
      <p className="font-medium">{label}</p>
      <div className="mt-2 grid gap-1">
        {Object.entries(items).map(([name, count]) => (
          <button className={`facet-row ${active.includes(name) ? "active" : ""}`} key={name} onClick={() => onToggle(name)}>
            <span>{name}</span>
            <strong>{count}</strong>
          </button>
        ))}
      </div>
    </div>
  );
}
