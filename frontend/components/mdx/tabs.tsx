"use client";

import { Children, ReactElement, ReactNode, isValidElement, useState } from "react";

export function Tab({ children }: { title?: string; children: ReactNode }) {
  return <>{children}</>;
}

export function Tabs({ children }: { children: ReactNode }) {
  const tabs = Children.toArray(children).filter(isValidElement) as ReactElement<{ title?: string; children: ReactNode }>[];
  const [active, setActive] = useState(0);
  if (tabs.length === 0) return null;
  return (
    <div className="mdx-tabs">
      <div className="mdx-tabs__bar" role="tablist">
        {tabs.map((tab, i) => (
          <button
            key={i}
            role="tab"
            aria-selected={i === active}
            className={`mdx-tabs__tab${i === active ? " is-active" : ""}`}
            onClick={() => setActive(i)}
            type="button"
          >
            {tab.props.title ?? `Tab ${i + 1}`}
          </button>
        ))}
      </div>
      <div className="mdx-tabs__panel" role="tabpanel">
        {tabs[active]}
      </div>
    </div>
  );
}
