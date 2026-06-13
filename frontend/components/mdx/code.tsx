"use client";

import { Children, ReactElement, ReactNode, isValidElement, useRef, useState } from "react";
import { Copy, Check } from "lucide-react";
import { Mermaid } from "./mermaid";
import { Kroki, isKrokiLang } from "./kroki";

function extractText(node: unknown): string {
  if (node == null) return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(extractText).join("");
  if (typeof node === "object" && "props" in (node as any)) {
    return extractText((node as any).props?.children);
  }
  return "";
}

type Parsed = { lang: string; filename: string; text: string; codeEl: ReactNode };

function parseCodeEl(codeEl: any): Parsed {
  const className: string = codeEl?.props?.className || "";
  const lang = (className.match(/language-([\w-]+)/) || [])[1] || "";
  const meta: string = codeEl?.props?.["data-meta"] || "";
  const token = meta.split(/\s+/).find((t) => t && !t.includes("=")) || "";
  return { lang, filename: token, text: extractText(codeEl), codeEl };
}

function CopyButton({ text }: { text: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      className="mdx-code__copy"
      aria-label="复制代码"
      onClick={() => {
        navigator.clipboard.writeText(text).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        });
      }}
    >
      {done ? <Check size={14} /> : <Copy size={14} />}
    </button>
  );
}

export function CodeBlock({
  lang,
  filename,
  text,
  grouped,
  children
}: {
  lang?: string;
  filename?: string;
  text: string;
  grouped?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="mdx-code">
      {!grouped && (filename || lang) ? (
        <div className="mdx-code__head">
          <span className="mdx-code__filename">{filename || lang}</span>
          {filename && lang ? <span className="mdx-code__lang">{lang}</span> : null}
        </div>
      ) : null}
      <div className="mdx-code__inner">
        <pre className={`mdx-code__pre${lang ? ` language-${lang}` : ""}`}>{children}</pre>
        <CopyButton text={text} />
      </div>
    </div>
  );
}

// Override for fenced code blocks (the <pre> element).
export function Pre({ children }: { children?: ReactNode }) {
  const codeEl = Children.toArray(children).find(isValidElement) as ReactElement | undefined;
  const parsed = parseCodeEl(codeEl);
  if (parsed.lang === "mermaid") {
    return <Mermaid chart={parsed.text} />;
  }
  if (isKrokiLang(parsed.lang)) {
    return <Kroki lang={parsed.lang} code={parsed.text} />;
  }
  return (
    <CodeBlock lang={parsed.lang} filename={parsed.filename} text={parsed.text}>
      {parsed.codeEl}
    </CodeBlock>
  );
}

export function CodeGroup({ children }: { children: ReactNode }) {
  const items = Children.toArray(children).filter(isValidElement) as ReactElement<{ children?: ReactNode }>[];
  const parsed = items.map((child) => parseCodeEl(Children.toArray(child.props.children).find(isValidElement)));
  const [active, setActive] = useState(0);
  if (parsed.length === 0) return null;
  return (
    <div className="mdx-code-group">
      <div className="mdx-code-group__bar" role="tablist">
        {parsed.map((p, i) => (
          <button
            key={i}
            type="button"
            role="tab"
            aria-selected={i === active}
            className={`mdx-code-group__tab${i === active ? " is-active" : ""}`}
            onClick={() => setActive(i)}
          >
            {p.filename || p.lang || `Tab ${i + 1}`}
          </button>
        ))}
      </div>
      <CodeBlock key={active} lang={parsed[active].lang} filename={parsed[active].filename} text={parsed[active].text} grouped>
        {parsed[active].codeEl}
      </CodeBlock>
    </div>
  );
}
