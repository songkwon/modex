import "katex/dist/katex.min.css";
import { compileMDX } from "next-mdx-remote/rsc";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeSlug from "rehype-slug";
import rehypeKatex from "rehype-katex";
import rehypeHighlight from "rehype-highlight";
import { mdxComponents } from "./index";
import { remarkCodeMeta, remarkGithubAlerts, rehypeToc } from "./remark-plugins";
import { MdxConfigProvider } from "./mdx-config";
import { getDocsPluginConfig, type PluginConfig } from "@/lib/api";

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

const on = (cfg: PluginConfig, key: string) => !(key in cfg) || cfg[key].enabled;

export async function MdxContent({ source }: { source: string }) {
  const plugins = await loadPluginConfig();
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
      components: mdxComponents,
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
        <div className="mdx">{content}</div>
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
