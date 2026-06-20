"use client";

import { ReactNode } from "react";
import { useI18n } from "@/lib/i18n";

// Frame wraps media with a border + optional caption.
export function Frame({ caption, children }: { caption?: string; children: ReactNode }) {
  return (
    <figure className="mdx-frame">
      <div className="mdx-frame__inner">{children}</div>
      {caption ? <figcaption className="mdx-frame__caption">{caption}</figcaption> : null}
    </figure>
  );
}

// Panel renders a bordered sidebar/aside block.
export function Panel({ children }: { children: ReactNode }) {
  return <aside className="mdx-panel">{children}</aside>;
}

// Update highlights a changelog entry with a label rail.
export function Update({ label, description, children }: { label?: string; description?: string; children: ReactNode }) {
  return (
    <div className="mdx-update">
      <div className="mdx-update__rail">
        {label ? <span className="mdx-update__label">{label}</span> : null}
        {description ? <span className="mdx-update__desc">{description}</span> : null}
      </div>
      <div className="mdx-update__body">{children}</div>
    </div>
  );
}

// Banner is a full-width announcement strip.
export function Banner({ children }: { children: ReactNode }) {
  return <div className="mdx-banner">{children}</div>;
}

// Snippet is the fallback for <Snippet name="…"/> tags that survive the
// pre-compile expansion (unknown name, or the snippets plugin is disabled).
// It renders nothing so a missing partial never breaks the page.
export function Snippet(_: { name?: string; children?: ReactNode }) {
  return null;
}

// Shows the authoring source next to the rendered result. This is intentionally
// simple so the docs example page can teach every supported Markdown/MDX shape.
export function MarkdownExample({
  title,
  source,
  children,
}: {
  title?: string;
  source: string;
  children: ReactNode;
}) {
  const { t } = useI18n();
  return (
    <div className="mdx-example">
      {title ? <div className="mdx-example__title">{title}</div> : null}
      <div className="mdx-example__grid">
        <div className="mdx-example__pane">
          <div className="mdx-example__label">{t("component.mdx.blocks.source_code")}</div>
          <pre className="mdx-example__source">
            <code>{source.trim()}</code>
          </pre>
        </div>
        <div className="mdx-example__pane">
          <div className="mdx-example__label">{t("component.mdx.blocks.preview")}</div>
          <div className="mdx-example__preview">{children}</div>
        </div>
      </div>
    </div>
  );
}
