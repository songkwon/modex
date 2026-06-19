"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Clock3, ExternalLink } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state";
import { recentDocs, syncedRecentDocs, type RecentDoc } from "@/lib/local-docs";

function formatViewedAt(value: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export default function RecentPage() {
  const [items, setItems] = useState<RecentDoc[]>([]);

  useEffect(() => {
    setItems(recentDocs());
    syncedRecentDocs().then(setItems);
  }, []);

  return (
    <main className="main">
      <section className="hero-panel compact-hero">
        <span className="hero-eyebrow">Personal</span>
        <h1 className="hero-title">最近访问</h1>
        <p className="hero-copy">最近阅读过的文档页按时间汇总。</p>
      </section>

      <section className="mt-5 table-card">
        {items.length === 0 ? (
          <EmptyState icon={Clock3} title="还没有访问记录" hint="最近阅读过的页面会出现在这里。" />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>文档</th>
                  <th>模块</th>
                  <th>版本</th>
                  <th>访问时间</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.doc_id}>
                    <td>
                      <Link href={item.href} className="font-medium">{item.title}</Link>
                      <div className="muted text-xs">{item.doc_id}</div>
                    </td>
                    <td>{item.module_name || item.module_key}</td>
                    <td><span className="tag">{item.docs_version}</span></td>
                    <td>{formatViewedAt(item.viewed_at)}</td>
                    <td>
                      <Link className="button icon-button" href={item.href} aria-label="打开文档">
                        <ExternalLink size={15} />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  );
}
