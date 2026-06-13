import { compileMDX } from "next-mdx-remote/rsc";
import remarkGfm from "remark-gfm";
import rehypeSlug from "rehype-slug";
import rehypeHighlight from "rehype-highlight";
import { mdxComponents } from "./index";

// remark plugin: carry a fenced code block's "meta" (e.g. ```ts config.ts) onto
// the <code> element so the renderer can show a filename header.
function remarkCodeMeta() {
  const walk = (node: any) => {
    if (!node || typeof node !== "object") return;
    if (node.type === "code" && node.meta) {
      node.data = node.data || {};
      node.data.hProperties = { ...(node.data.hProperties || {}), "data-meta": node.meta };
    }
    if (Array.isArray(node.children)) node.children.forEach(walk);
  };
  return (tree: any) => walk(tree);
}

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
          remarkPlugins: [remarkGfm, remarkCodeMeta],
          rehypePlugins: [rehypeSlug, [rehypeHighlight, { ignoreMissing: true, detect: true }]]
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
