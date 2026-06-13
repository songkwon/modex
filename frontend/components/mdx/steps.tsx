import { Children, ReactElement, ReactNode, isValidElement } from "react";
import { Icon } from "./icon";

export function Step({ title, icon, children }: { title?: string; icon?: string; children: ReactNode }) {
  // Rendering handled by Steps so each step gets its sequential marker.
  return (
    <div className="mdx-step">
      <div className="mdx-step__head">
        {icon ? <Icon icon={icon} size={14} /> : null}
        {title ? <p className="mdx-step__title">{title}</p> : null}
      </div>
      <div className="mdx-step__body">{children}</div>
    </div>
  );
}

export function Steps({ children }: { children: ReactNode }) {
  const steps = Children.toArray(children).filter(isValidElement) as ReactElement[];
  return (
    <div className="mdx-steps">
      {steps.map((step, i) => (
        <div className="mdx-steps__row" key={i}>
          <div className="mdx-steps__marker" aria-hidden>
            {i + 1}
          </div>
          <div className="mdx-steps__content">{step}</div>
        </div>
      ))}
    </div>
  );
}
