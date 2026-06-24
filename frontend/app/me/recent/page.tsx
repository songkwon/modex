"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Clock3, ExternalLink } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state";
import { recentDocs, syncedRecentDocs, type RecentDoc } from "@/lib/local-docs";
import { useI18n } from "@/lib/i18n";

function formatViewedAt(value: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export default function RecentPage() {
  const { t } = useI18n();
  const [items, setItems] = useState<RecentDoc[]>([]);

  useEffect(() => {
    setItems(recentDocs());
    syncedRecentDocs().then(setItems);
  }, []);

  return (
    <main className="main">
      <section className="hero-panel compact-hero">
        <span className="hero-eyebrow">Personal</span>
        <h1 className="hero-title">{t("me.recent.recently_accessed")}</h1>
        <p className="hero-copy">{t("me.recent.recently_viewed_document_pages_grouped_by_time")}</p>
      </section>

      <section className="mt-5 table-card">
        {items.length === 0 ? (
          <EmptyState icon={Clock3} title={t("me.recent.no_access_records_yet")} hint={t("me.recent.recently_viewed_pages_appear_here")} />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("me.recent.documentation")}</th>
                  <th>{t("me.recent.module")}</th>
                  <th>{t("me.recent.version")}</th>
                  <th>{t("me.recent.access_time")}</th>
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
                    <td>{item.module_name || "-"}</td>
                    <td><span className="tag">{item.docs_version}</span></td>
                    <td>{formatViewedAt(item.viewed_at)}</td>
                    <td>
                      <Link className="button icon-button" href={item.href} aria-label={t("me.recent.open_document")}>
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
