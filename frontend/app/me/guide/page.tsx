import { readFile } from "node:fs/promises";
import path from "node:path";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import { MdxContent } from "@/components/mdx/mdx-content";
import { DocToc } from "@/components/doc-toc";
import { DocSourceToggle } from "@/components/doc-source-toggle";
import { getServerI18n } from "@/lib/i18n-server";
import { publicApiBaseURL } from "@/lib/runtime-config";
import type { User } from "@/types/modex";

async function loadGuide() {
  return readFile(path.join(process.cwd(), "content/modex-guide.mdx"), "utf8");
}

async function isSuperAdminRequest() {
  const requestHeaders = await headers();
  const cookie = requestHeaders.get("cookie") || "";
  const baseURL = process.env.INTERNAL_API_BASE_URL || publicApiBaseURL();
  try {
    const res = await fetch(`${baseURL}/api/auth/me?optional=1`, {
      headers: cookie ? { cookie } : undefined,
      cache: "no-store",
    });
    if (!res.ok) return false;
    const user = await res.json() as User | null;
    return !!user?.is_super_admin;
  } catch {
    return false;
  }
}

export default async function ModexGuidePage() {
  if (!(await isSuperAdminRequest())) {
    redirect("/");
  }
  const source = await loadGuide();
  const { t } = await getServerI18n();
  return (
    <main className="main docs-shell">
      <section className="doc-layout doc-layout--single">
        <aside className="doc-sidebar" />
        <article className="panel prose doc-page">
          <p className="muted text-sm doc-breadcrumb">Modex / Guide</p>
          <div className="doc-title-row">
            <h1 className="doc-title" id="overview">{t("me.guide.modex_project_guide")}</h1>
          </div>
          <DocSourceToggle source={source}>
            <MdxContent source={source} />
          </DocSourceToggle>
        </article>
        <aside className="doc-toc">
          <div className="doc-toc-card">
            <DocToc />
          </div>
        </aside>
      </section>
    </main>
  );
}
