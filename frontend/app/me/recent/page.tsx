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
        <h1 className="hero-title">{t("legacy.de314b445e07")}</h1>
        <p className="hero-copy">{t("legacy.648db0493044")}</p>
      </section>

      <section className="mt-5 table-card">
        {items.length === 0 ? (
          <EmptyState icon={Clock3} title={t("legacy.5751457093ee")} hint={t("legacy.6987b70b304c")} />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("legacy.2687ccdbb1d2")}</th>
                  <th>{t("legacy.b07e5088eafa")}</th>
                  <th>{t("legacy.5f76b2bf82dd")}</th>
                  <th>{t("legacy.92cfc7de2e88")}</th>
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
                      <Link className="button icon-button" href={item.href} aria-label={t("legacy.d1961d380a4d")}>
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
