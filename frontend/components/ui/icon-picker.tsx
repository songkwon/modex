"use client";

import {
  Book, BookOpen, Box, Boxes, Cpu, Database, FileText, FolderTree, GitBranch,
  Layers, Layout, type LucideIcon, Package, Rocket, Settings, Shield, Terminal,
  Wrench, Workflow, Zap,
} from "lucide-react";

// Named icon catalog shared by the picker and anywhere a category icon is shown.
export const ICON_MAP: Record<string, LucideIcon> = {
  wrench: Wrench,
  package: Package,
  box: Box,
  boxes: Boxes,
  layers: Layers,
  layout: Layout,
  "book-open": BookOpen,
  book: Book,
  cpu: Cpu,
  "git-branch": GitBranch,
  tool: Wrench,
  "file-text": FileText,
  database: Database,
  terminal: Terminal,
  workflow: Workflow,
  rocket: Rocket,
  shield: Shield,
  zap: Zap,
  settings: Settings,
};

export function CategoryIcon({ name, size = 15 }: { name?: string; size?: number }) {
  const Icon = (name && ICON_MAP[name]) || FolderTree;
  return <Icon size={size} />;
}

/** colorpad-style grid: every icon is rendered so the user picks visually. */
export function IconPicker({ value, onChange }: { value?: string; onChange: (name: string) => void }) {
  return (
    <div className="icon-picker">
      {Object.entries(ICON_MAP).map(([name, Icon]) => (
        <button
          type="button"
          key={name}
          title={name}
          className={`icon-swatch${value === name ? " active" : ""}`}
          onClick={() => onChange(value === name ? "" : name)}
        >
          <Icon size={18} />
        </button>
      ))}
    </div>
  );
}
