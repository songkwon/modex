import { ReactNode } from "react";

// Tree renders a file/folder hierarchy. Content is authored as a nested
// markdown list, styled here to look like a directory tree.
export function Tree({ children }: { children: ReactNode }) {
  return <div className="mdx-tree">{children}</div>;
}

export function Folder({ name, children }: { name?: string; children?: ReactNode }) {
  return (
    <div className="mdx-tree__folder">
      <span className="mdx-tree__label">{name}</span>
      {children ? <div className="mdx-tree__children">{children}</div> : null}
    </div>
  );
}

export function File({ name }: { name?: string }) {
  return <div className="mdx-tree__file">{name}</div>;
}
