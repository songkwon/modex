import "katex/dist/katex.min.css";
import { compileMDX } from "next-mdx-remote/rsc";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeSlug from "rehype-slug";
import rehypeKatex from "rehype-katex";
import rehypeHighlight from "rehype-highlight";
import { mdxComponents } from "./index";
import { remarkCodeMeta, remarkGithubAlerts, rehypeToc } from "./remark-plugins";

export async function MdxContent({ source }: { source: string }) {
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
        mdxOptions: {
          remarkPlugins: [remarkGfm, remarkMath, remarkGithubAlerts, remarkCodeMeta],
          // rehypeToc runs after rehypeSlug so heading ids exist; rehypeKatex
          // turns remark-math nodes into rendered formulas.
          rehypePlugins: [rehypeSlug, rehypeToc, rehypeKatex, [rehypeHighlight, { ignoreMissing: true, detect: true }]]
        }
      }
    });
    return <div className="mdx">{content}</div>;
  } catch (err) {
    return (
      <div className="mdx mdx--error">
        <p className="muted text-sm">文档渲染失败，已回退为纯文本。</p>
        <pre className="mdx-code__pre">{source}</pre>
      </div>
    );
  }
}
