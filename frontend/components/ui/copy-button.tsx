"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";

type Props = {
  value: string;
  title?: string;
  className?: string;
  size?: number;
  label?: string;
};

/** Icon button that copies `value` and briefly swaps to a checkmark. */
export function CopyButton({ value, title = "复制", className = "icon-btn", size = 14, label }: Props) {
  const [copied, setCopied] = useState(false);
  async function copy() {
    try {
      await navigator.clipboard?.writeText(value);
    } catch {
      /* clipboard may be unavailable (insecure context) */
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }
  return (
    <button
      type="button"
      className={className}
      title={title}
      aria-label={title}
      onClick={copy}
      data-copied={copied || undefined}
    >
      {copied ? <Check size={size} /> : <Copy size={size} />}
      {label ? <span>{copied ? "已复制" : label}</span> : null}
    </button>
  );
}
