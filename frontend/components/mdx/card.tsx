import { ReactNode } from "react";
import { ArrowRight } from "lucide-react";
import { Icon } from "./icon";

export function Card({
  title,
  icon,
  href,
  horizontal,
  children
}: {
  title?: string;
  icon?: string;
  href?: string;
  horizontal?: boolean;
  children?: ReactNode;
}) {
  const inner = (
    <>
      {icon ? (
        <div className="mdx-card__icon">
          <Icon icon={icon} size={20} />
        </div>
      ) : null}
      <div className="mdx-card__main">
        {title ? <p className="mdx-card__title">{title}</p> : null}
        {children ? <div className="mdx-card__body">{children}</div> : null}
      </div>
      {href ? <ArrowRight size={16} className="mdx-card__arrow" aria-hidden /> : null}
    </>
  );
  const cls = `mdx-card${horizontal ? " mdx-card--horizontal" : ""}${href ? " mdx-card--link" : ""}`;
  if (href) {
    return (
      <a className={cls} href={href}>
        {inner}
      </a>
    );
  }
  return <div className={cls}>{inner}</div>;
}

export function CardGroup({ cols = 2, children }: { cols?: number; children: ReactNode }) {
  return (
    <div className="mdx-card-group" style={{ "--mdx-cols": cols } as React.CSSProperties}>
      {children}
    </div>
  );
}

// Columns is a layout primitive; visually identical grid to CardGroup.
export function Columns({ cols = 2, children }: { cols?: number; children: ReactNode }) {
  return (
    <div className="mdx-card-group" style={{ "--mdx-cols": cols } as React.CSSProperties}>
      {children}
    </div>
  );
}

// Tile is a clickable grid cell, an alias of Card.
export const Tile = Card;
