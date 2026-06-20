import "katex/dist/katex.min.css";
import type { ImgHTMLAttributes, ReactElement } from "react";
import { compileMDX } from "next-mdx-remote/rsc";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeSlug from "rehype-slug";
import rehypeKatex from "rehype-katex";
import rehypeHighlight from "rehype-highlight";
import rehypeRaw from "rehype-raw";
import bash from "highlight.js/lib/languages/bash";
import c from "highlight.js/lib/languages/c";
import cpp from "highlight.js/lib/languages/cpp";
import csharp from "highlight.js/lib/languages/csharp";
import css from "highlight.js/lib/languages/css";
import delphi from "highlight.js/lib/languages/delphi";
import dockerfile from "highlight.js/lib/languages/dockerfile";
import go from "highlight.js/lib/languages/go";
import java from "highlight.js/lib/languages/java";
import javascript from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import markdown from "highlight.js/lib/languages/markdown";
import python from "highlight.js/lib/languages/python";
import rust from "highlight.js/lib/languages/rust";
import shell from "highlight.js/lib/languages/shell";
import sql from "highlight.js/lib/languages/sql";
import typescript from "highlight.js/lib/languages/typescript";
import xml from "highlight.js/lib/languages/xml";
import yaml from "highlight.js/lib/languages/yaml";
import { getServerI18n } from "@/lib/i18n-server";
import { mdxComponents } from "./index";
import { remarkCodeMeta, remarkGithubAlerts, rehypeToc } from "./remark-plugins";
import { MdxConfigProvider, UploadedFencesProvider } from "./mdx-config";
import { SandboxedPlugin } from "./sandboxed-plugin";
import { serializableProps } from "./plugin-utils";
import { expandSnippets } from "./snippets";
import { getDocsPluginConfig, getDocsSnippets, getDocsUploadedPlugins, type PluginConfig, type UploadedPlugin } from "@/lib/api";

// Effective plugin config drives which plugins run. On any failure we fall back
// to "all enabled" (empty config → pluginEnabled returns its true default).
async function loadPluginConfig(): Promise<PluginConfig> {
  try {
    const cfg = await getDocsPluginConfig();
    return cfg.plugins ?? {};
  } catch {
    return {};
  }
}

// Snippet library + variables, keyed for expansion. Empty on failure.
async function loadSnippets(): Promise<{ snippets: Record<string, string>; vars: Record<string, string> }> {
  try {
    const data = await getDocsSnippets();
    const snippets: Record<string, string> = {};
    for (const s of data.snippets || []) snippets[s.key] = s.content;
    return { snippets, vars: data.variables || {} };
  } catch {
    return { snippets: {}, vars: {} };
  }
}

// Enabled uploaded plugins for the renderer. Empty on failure.
async function loadUploadedPlugins(): Promise<UploadedPlugin[]> {
  try {
    return (await getDocsUploadedPlugins()).plugins || [];
  } catch {
    return [];
  }
}

const on = (cfg: PluginConfig, key: string) => !(key in cfg) || cfg[key].enabled;

export async function MdxContent({ source, assetBase }: { source: string; assetBase?: string }) {
  const { t } = await getServerI18n();
  // Drop a leading Confluence-style [TOC]/[[toc]] macro (and stray leading
  // <br/>) — the right-rail "本页目录" replaces an inline table of contents.
  source = source
    .replace(/^﻿/, "")
    .replace(/^\s+/, "")
    .replace(/^(?:<br\s*\/?>\s*)+/i, "")
    .replace(/^\[\[?toc\]?\]\s*/i, "");
  source = rewriteMarkdownAssetURLs(source, assetBase);

  const plugins = await loadPluginConfig();
  if (on(plugins, "snippets")) {
    const { snippets, vars } = await loadSnippets();
    source = expandSnippets(source, snippets, vars);
  }

  // Uploaded plugins: component-kind become dynamic MDX tags rendered in a
  // sandboxed iframe; fence-kind are routed by <Pre> via the fences context.
  const uploaded = await loadUploadedPlugins();
  const dynamicComponents: Record<string, (p: Record<string, unknown>) => ReactElement> = {};
  const uploadedFences: Record<string, string> = {};
  for (const up of uploaded) {
    if (up.kind === "component" && up.tag) {
      const codeStr = up.code;
      dynamicComponents[up.tag] = (p: Record<string, unknown>) => (
        <SandboxedPlugin code={codeStr} props={serializableProps(p)} />
      );
    } else if (up.kind === "fence" && up.lang) {
      uploadedFences[up.lang] = up.code;
    }
  }
  const remarkPlugins: any[] = [
    remarkGfm,
    ...(on(plugins, "math") ? [remarkMath] : []),
    ...(on(plugins, "github_alerts") ? [remarkGithubAlerts] : []),
    remarkCodeMeta
  ];
  const rehypePlugins: any[] = [
    rehypeSlug,
    ...(on(plugins, "toc") ? [rehypeToc] : []),
    ...(on(plugins, "math") ? [rehypeKatex] : []),
    [
      rehypeHighlight,
      {
        ignoreMissing: true,
        detect: true,
        languages: {
          bash,
          c,
          cpp,
          csharp,
          css,
          delphi,
          dockerfile,
          go,
          java,
          javascript,
          js: javascript,
          json,
          markdown,
          md: markdown,
          mdx: markdown,
          pascal: delphi,
          python,
          py: python,
          rust,
          rs: rust,
          shell,
          sh: shell,
          sql,
          typescript,
          ts: typescript,
          tsx: typescript,
          xml,
          html: xml,
          yaml,
          yml: yaml
        }
      }
    ]
  ];

  const compile = (format: "mdx" | "md") =>
    compileMDX({
      source,
      components: { ...mdxComponents, ...dynamicComponents, img: imageComponent(assetBase) },
      options: {
        parseFrontmatter: true,
        // Mintlify-style components rely on JSX expression props (cols={2}) and
        // `{frontmatter.x}` expressions, so we must allow JS expressions. We keep
        // blockDangerousJS on, which still rejects eval/Function/require/process
        // and other escape hatches at compile time.
        blockJS: false,
        blockDangerousJS: true,
        // format "md" parses as plain CommonMark: `{...}` and `<...>` (e.g. Pascal
        // braces, generics) are literal text, not JSX. We try MDX first so authored
        // docs keep their components, then fall back to md for imported plain
        // Markdown that isn't MDX-safe — rather than dumping raw source. In md mode
        // we allow raw HTML and run rehype-raw so stray angle-bracket content
        // (Delphi generics, <br/>) is preserved instead of silently dropped.
        mdxOptions:
          format === "md"
            ? { format, remarkRehypeOptions: { allowDangerousHtml: true }, remarkPlugins, rehypePlugins: [rehypeRaw, ...rehypePlugins] }
            : { format, remarkPlugins, rehypePlugins }
      }
    });

  let content: Awaited<ReturnType<typeof compileMDX>>["content"] | null = null;
  try {
    content = (await compile("mdx")).content;
  } catch {
    try {
      content = (await compile("md")).content;
    } catch {
      return (
        <div className="mdx mdx--error">
          <p className="muted text-sm">{t("legacy.90394e52e026")}</p>
          <pre className="mdx-code__pre">{source}</pre>
        </div>
      );
    }
  }
  return (
    <MdxConfigProvider value={plugins}>
      <UploadedFencesProvider value={uploadedFences}>
        <div className="mdx">{content}</div>
      </UploadedFencesProvider>
    </MdxConfigProvider>
  );
}

const PUBLIC_API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8671";

function imageComponent(assetBase?: string) {
  return function Image(props: ImgHTMLAttributes<HTMLImageElement>) {
    const src = typeof props.src === "string" ? resolveAssetURL(props.src, assetBase) : props.src;
    return <img {...props} src={src} />;
  };
}

function rewriteMarkdownAssetURLs(source: string, assetBase?: string) {
  return source
    .replace(/(!\[[^\]]*\]\()([^)]+)(\))/g, (_m, open: string, raw: string, close: string) => {
      return `${open}${resolveAssetURL(raw.trim(), assetBase)}${close}`;
    })
    .replace(/(<img\b[^>]*\bsrc=["'])([^"']+)(["'][^>]*>)/gi, (_m, open: string, raw: string, close: string) => {
      return `${open}${resolveAssetURL(raw.trim(), assetBase)}${close}`;
    });
}

function resolveAssetURL(src: string, assetBase?: string) {
  const lower = src.toLowerCase();
  if (!src || lower.startsWith("http://") || lower.startsWith("https://") || lower.startsWith("data:") || src.startsWith("//")) {
    return src;
  }
  if (src.startsWith("/api/docs/")) {
    return `${PUBLIC_API_BASE}${src}`;
  }
  if (src.startsWith("/")) {
    return src;
  }
  if (!assetBase) {
    return src;
  }
  const base = assetBase.endsWith("/") ? assetBase : `${assetBase}/`;
  return new URL(src.replace(/^\.\/+/, ""), base).toString();
}
