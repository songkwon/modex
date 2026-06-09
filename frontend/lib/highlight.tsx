import type { ReactNode } from "react";

// highlight wraps matched terms (case-insensitive) in <mark>, CJK-safe.
export function highlight(text: string, terms: string[] = []): ReactNode {
  if (!text || terms.length === 0) return text;
  const escaped = terms.filter(Boolean).map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  if (escaped.length === 0) return text;
  const re = new RegExp(`(${escaped.join("|")})`, "gi");
  const parts = text.split(re);
  return parts.map((part, i) =>
    re.test(part) ? <mark key={i} className="ds-mark">{part}</mark> : <span key={i}>{part}</span>
  );
}
