"use client";

import { useMemo, useState, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";
import type { AskResponse } from "@/types/modex";

type Segment =
  | { type: "code"; lang: string; text: string }
  | { type: "heading"; level: number; text: string }
  | { type: "list"; ordered: boolean; items: string[] }
  | { type: "paragraph"; text: string };

export function splitAnswerParts(answer: Pick<AskResponse, "answer"> & Partial<AskResponse>) {
  const raw = answer.answer || "";
  const explicitReasoning = firstNonEmpty(answer.reasoning, answer.thinking, answer.reasoning_content);
  const thinkBlocks: string[] = [];
  const body = raw.replace(/<think>([\s\S]*?)<\/think>/gi, (_m, content: string) => {
    if (content.trim()) thinkBlocks.push(content.trim());
    return "";
  }).trim();

  return {
    reasoning: firstNonEmpty(explicitReasoning, thinkBlocks.join("\n\n")),
    answer: body || raw
  };
}

export function AiMarkdown({
  answer,
  reasoning,
  compact = false
}: {
  answer: string;
  reasoning?: string;
  compact?: boolean;
}) {
  return (
    <div className={`ai-md ${compact ? "ai-md--compact" : ""}`}>
      {reasoning ? <ReasoningBlock text={reasoning} /> : null}
      <MarkdownText text={answer} />
    </div>
  );
}

function ReasoningBlock({ text }: { text: string }) {
  const [open, setOpen] = useState(false);
  return (
    <section className={`ai-reasoning ${open ? "open" : ""}`}>
      <button className="ai-reasoning__toggle" type="button" onClick={() => setOpen((v) => !v)}>
        <ChevronDown size={15} />
        <span>思考过程</span>
      </button>
      {open ? <div className="ai-reasoning__body"><MarkdownText text={text} /></div> : null}
    </section>
  );
}

function MarkdownText({ text }: { text: string }) {
  const segments = useMemo(() => parseMarkdown(text), [text]);
  return (
    <div className="ai-md__content">
      {segments.map((segment, index) => renderSegment(segment, index))}
    </div>
  );
}

function parseMarkdown(source: string): Segment[] {
  const lines = source.replace(/\r\n/g, "\n").split("\n");
  const segments: Segment[] = [];
  let paragraph: string[] = [];
  let list: { ordered: boolean; items: string[] } | null = null;

  const flushParagraph = () => {
    if (!paragraph.length) return;
    segments.push({ type: "paragraph", text: paragraph.join("\n").trim() });
    paragraph = [];
  };
  const flushList = () => {
    if (!list) return;
    segments.push({ type: "list", ordered: list.ordered, items: list.items });
    list = null;
  };

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i];
    const fence = line.match(/^```([A-Za-z0-9_-]+)?\s*$/);
    if (fence) {
      flushParagraph();
      flushList();
      const code: string[] = [];
      i += 1;
      while (i < lines.length && !/^```\s*$/.test(lines[i])) {
        code.push(lines[i]);
        i += 1;
      }
      segments.push({ type: "code", lang: fence[1] || "", text: code.join("\n") });
      continue;
    }

    if (!line.trim()) {
      flushParagraph();
      flushList();
      continue;
    }

    const heading = line.match(/^(#{1,4})\s+(.+)$/);
    if (heading) {
      flushParagraph();
      flushList();
      segments.push({ type: "heading", level: heading[1].length, text: heading[2].trim() });
      continue;
    }

    const unordered = line.match(/^\s*[-*]\s+(.+)$/);
    const ordered = line.match(/^\s*\d+[.)]\s+(.+)$/);
    if (unordered || ordered) {
      flushParagraph();
      const orderedList = Boolean(ordered);
      if (!list || list.ordered !== orderedList) flushList();
      list ||= { ordered: orderedList, items: [] };
      list.items.push((ordered?.[1] || unordered?.[1] || "").trim());
      continue;
    }

    flushList();
    paragraph.push(line);
  }

  flushParagraph();
  flushList();
  return segments;
}

function renderSegment(segment: Segment, index: number) {
  switch (segment.type) {
    case "code":
      return (
        <pre className="ai-md__pre" key={index}>
          {segment.lang ? <span className="ai-md__lang">{segment.lang}</span> : null}
          <code>{segment.text}</code>
        </pre>
      );
    case "heading":
      if (segment.level === 1) return <h3 key={index}>{renderInline(segment.text)}</h3>;
      if (segment.level === 2) return <h4 key={index}>{renderInline(segment.text)}</h4>;
      if (segment.level === 3) return <h5 key={index}>{renderInline(segment.text)}</h5>;
      return <h6 key={index}>{renderInline(segment.text)}</h6>;
    case "list": {
      const Tag = segment.ordered ? "ol" : "ul";
      return <Tag key={index}>{segment.items.map((item, i) => <li key={i}>{renderInline(item)}</li>)}</Tag>;
    }
    case "paragraph":
      return <p key={index}>{renderInline(segment.text)}</p>;
  }
}

function renderInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  const pattern = /(`[^`]+`|\*\*[^*]+\*\*|\[[^\]]+\]\([^)]+\))/g;
  let last = 0;
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(text))) {
    if (match.index > last) nodes.push(text.slice(last, match.index));
    const token = match[0];
    if (token.startsWith("`")) {
      nodes.push(<code key={nodes.length}>{token.slice(1, -1)}</code>);
    } else if (token.startsWith("**")) {
      nodes.push(<strong key={nodes.length}>{token.slice(2, -2)}</strong>);
    } else {
      const link = token.match(/^\[([^\]]+)\]\(([^)]+)\)$/);
      const href = link?.[2] || "";
      nodes.push(<a key={nodes.length} href={safeHref(href)} target="_blank" rel="noreferrer">{link?.[1] || href}</a>);
    }
    last = pattern.lastIndex;
  }
  if (last < text.length) nodes.push(text.slice(last));
  return nodes;
}

function safeHref(href: string) {
  if (/^(https?:|mailto:|\/)/i.test(href)) return href;
  return "#";
}

function firstNonEmpty(...values: Array<string | undefined>) {
  return values.map((v) => v?.trim()).find(Boolean) || "";
}
