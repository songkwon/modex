"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Check, ChevronDown } from "lucide-react";
import { getEntries } from "@/lib/api";

type VersionOption = { docs_version: string; display_name?: string };

// VersionSwitcher lets the reader switch the doc version. Navigating keeps the
// current entry key; if that entry doesn't exist in the target version the doc
// route resolves to its primary entry server-side.
export function VersionSwitcher({
  moduleKey,
  current,
  entryKey,
  versions,
}: {
  moduleKey: string;
  current: string;
  entryKey: string;
  versions: VersionOption[];
}) {
  const router = useRouter();
  const ref = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [switching, setSwitching] = useState(false);
  const options = versions.length ? versions : [{ docs_version: current, display_name: current }];
  const currentLabel = options.find((v) => v.docs_version === current)?.display_name || current;

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current?.contains(e.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  async function switchVersion(target: string) {
    if (!target || target === current || switching) return;
    setSwitching(true);
    setOpen(false);
    try {
      const targetEntries = await getEntries(moduleKey, target);
      const nextEntry = targetEntries.find((entry) => entry.entry_key === entryKey) || targetEntries[0];
      router.push(nextEntry ? `/docs/${moduleKey}/${target}/${nextEntry.entry_key}` : `/docs/${moduleKey}/${target}`);
    } finally {
      setSwitching(false);
    }
  }
  return (
    <div className="doc-version-switcher" ref={ref}>
      <button
        type="button"
        className="doc-version-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={switching}
        onClick={() => setOpen((v) => !v)}
      >
        <span>{currentLabel}</span>
        <ChevronDown size={13} />
      </button>
      {open ? (
        <div className="doc-version-menu" role="listbox">
          {options.map((v) => {
            const selected = v.docs_version === current;
            return (
              <button
                type="button"
                key={v.docs_version}
                className={`doc-version-option${selected ? " selected" : ""}`}
                role="option"
                aria-selected={selected}
                onClick={() => switchVersion(v.docs_version)}
              >
                <span>{v.display_name || v.docs_version}</span>
                {selected ? <Check size={14} /> : null}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
