import "katex/dist/katex.min.css";
import type { ReactElement } from "react";
import { compileMDX } from "next-mdx-remote/rsc";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeSlug from "rehype-slug";
import rehypeKatex from "rehype-katex";
import rehypeHighlight from "rehype-highlight";
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

export async function MdxContent({ source }: { source: string }) {
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
    [rehypeHighlight, { ignoreMissing: true, detect: true }]
  ];

  try {
    const { content } = await compileMDX({
      source,
      components: { ...mdxComponents, ...dynamicComponents },
      options: {
        parseFrontmatter: true,
        // Mintlify-style components rely on JSX expression props (cols={2}) and
        // `{frontmatter.x}` expressions, so we must allow JS expressions. We keep
        // blockDangerousJS on, which still rejects eval/Function/require/process
        // and other escape hatches at compile time.
        blockJS: false,
        blockDangerousJS: true,
        mdxOptions: { remarkPlugins, rehypePlugins }
      }
    });
    return (
      <MdxConfigProvider value={plugins}>
        <UploadedFencesProvider value={uploadedFences}>
          <div className="mdx">{content}</div>
        </UploadedFencesProvider>
      </MdxConfigProvider>
    );
  } catch (err) {
    return (
      <div className="mdx mdx--error">
        <p className="muted text-sm">文档渲染失败，已回退为纯文本。</p>
        <pre className="mdx-code__pre">{source}</pre>
      </div>
    );
  }
}
