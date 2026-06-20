"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { useI18n } from "@/lib/i18n";

type Props = {
  value: string;
  title?: string;
  className?: string;
  size?: number;
  label?: string;
};

/** Icon button that copies `value` and briefly swaps to a checkmark. */
export function CopyButton({ value, title, className = "icon-btn", size = 14, label }: Props) {
  const { t } = useI18n();
  const titleText = title ?? t("legacy.63d90d977348");
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
      title={titleText}
      aria-label={titleText}
      onClick={copy}
      data-copied={copied || undefined}
    >
      {copied ? <Check size={size} /> : <Copy size={size} />}
      {label ? <span>{copied ? t("legacy.8f6f8d979c98") : label}</span> : null}
    </button>
  );
}
