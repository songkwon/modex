"use client";

import { useEffect, useRef } from "react";

// ImmersiveChrome runs on the full-screen site-builder doc page. It keeps the
// Modex topbar (user menu, search, theme) available but auto-hides it a few
// seconds after load so the embedded site reads like a standalone site. The bar
// slides back in when the pointer reaches the top edge — detected via a thin
// reveal strip rendered above the iframe, because the iframe swallows pointer
// events over its own area.
const HIDE_DELAY = 2500;

export function ImmersiveChrome() {
  const zoneRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const root = document.documentElement;
    const topbar = document.querySelector<HTMLElement>(".topbar");
    const zone = zoneRef.current;
    let timer: number | undefined;

    const clear = () => {
      if (timer) {
        window.clearTimeout(timer);
        timer = undefined;
      }
    };
    const show = () => root.classList.remove("topbar-hidden");
    const scheduleHide = () => {
      clear();
      timer = window.setTimeout(() => {
        root.classList.add("topbar-hidden");
        timer = undefined;
      }, HIDE_DELAY);
    };
    const reveal = () => {
      show();
      clear();
    };

    root.classList.add("doc-immersive");
    scheduleHide();

    zone?.addEventListener("mouseenter", reveal);
    topbar?.addEventListener("mouseenter", reveal);
    topbar?.addEventListener("mouseleave", scheduleHide);

    return () => {
      clear();
      zone?.removeEventListener("mouseenter", reveal);
      topbar?.removeEventListener("mouseenter", reveal);
      topbar?.removeEventListener("mouseleave", scheduleHide);
      root.classList.remove("doc-immersive", "topbar-hidden");
    };
  }, []);

  return <div ref={zoneRef} className="topbar-reveal-zone" aria-hidden />;
}
