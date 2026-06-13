"use client";

import { ReactNode, useState } from "react";
import { Icon } from "./icon";

export function Tooltip({ tip, children }: { tip?: string; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <span
      className="mdx-tooltip"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
      tabIndex={0}
    >
      <span className="mdx-tooltip__trigger">{children}</span>
      {open && tip ? <span className="mdx-tooltip__bubble" role="tooltip">{tip}</span> : null}
    </span>
  );
}

export function Badge({ color, children }: { color?: string; children: ReactNode }) {
  return (
    <span className={`mdx-badge${color ? ` mdx-badge--${color}` : ""}`}>{children}</span>
  );
}

export function Color({ children }: { children: ReactNode }) {
  const value = typeof children === "string" ? children : "";
  return (
    <span className="mdx-color">
      <span className="mdx-color__swatch" style={{ backgroundColor: value }} aria-hidden />
      <code className="mdx-color__value">{value}</code>
    </span>
  );
}

// Inline icon component, e.g. <Icon icon="rocket" />
export function IconTag({ icon, size }: { icon?: string; size?: number }) {
  return <Icon icon={icon} size={size ?? 16} className="mdx-inline-icon" />;
}
