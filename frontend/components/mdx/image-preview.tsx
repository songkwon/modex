"use client";

import type { ImgHTMLAttributes } from "react";
import { useEffect, useState } from "react";
import { X } from "lucide-react";

export function MdxImagePreview(props: ImgHTMLAttributes<HTMLImageElement>) {
  const [open, setOpen] = useState(false);
  const src = typeof props.src === "string" ? props.src : "";
  const alt = typeof props.alt === "string" ? props.alt : "";

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("keydown", onKeyDown);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = "";
    };
  }, [open]);

  return (
    <>
      <button className="mdx-image-trigger" type="button" onClick={() => src && setOpen(true)}>
        <img {...props} alt={alt} />
      </button>
      {open ? (
        <div className="mdx-image-preview" role="dialog" aria-modal="true" aria-label={alt || "Image preview"} onClick={() => setOpen(false)}>
          <button className="mdx-image-preview__close" type="button" aria-label="Close image preview" onClick={() => setOpen(false)}>
            <X size={20} />
          </button>
          <img className="mdx-image-preview__img" src={src} alt={alt} onClick={(event) => event.stopPropagation()} />
        </div>
      ) : null}
    </>
  );
}
