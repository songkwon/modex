"use client";

import { ReactNode, useState } from "react";
import { ChevronRight } from "lucide-react";
import { Icon } from "./icon";
import { useI18n } from "@/lib/i18n";

export function Accordion({
  title,
  description,
  icon,
  defaultOpen = false,
  children
}: {
  title?: string;
  description?: string;
  icon?: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className={`mdx-accordion${open ? " is-open" : ""}`}>
      <button type="button" className="mdx-accordion__head" aria-expanded={open} onClick={() => setOpen((v) => !v)}>
        <ChevronRight size={16} className="mdx-accordion__chevron" aria-hidden />
        {icon ? <Icon icon={icon} size={16} className="mdx-accordion__icon" /> : null}
        <span className="mdx-accordion__title">{title}</span>
      </button>
      {open ? (
        <div className="mdx-accordion__body">
          {description ? <p className="mdx-accordion__desc">{description}</p> : null}
          {children}
        </div>
      ) : null}
    </div>
  );
}

export function AccordionGroup({ children }: { children: ReactNode }) {
  return <div className="mdx-accordion-group">{children}</div>;
}

// Expandable behaves like an inline accordion, typically nesting fields.
export function Expandable({ title, defaultOpen = false, children }: { title?: string; defaultOpen?: boolean; children: ReactNode }) {
  const { t } = useI18n();
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className={`mdx-expandable${open ? " is-open" : ""}`}>
      <button type="button" className="mdx-expandable__head" aria-expanded={open} onClick={() => setOpen((v) => !v)}>
        <ChevronRight size={14} className="mdx-accordion__chevron" aria-hidden />
        <span>{title ?? t("legacy.00bd3960fea8")}</span>
      </button>
      {open ? <div className="mdx-expandable__body">{children}</div> : null}
    </div>
  );
}
