"use client";

import { useEffect } from "react";
import { Copy, Check } from "lucide-react";

export function CodeBlockCopy() {
  useEffect(() => {
    const containers = document.querySelectorAll<HTMLElement>(".embedded-doc-html pre");
    containers.forEach((pre) => {
      if (pre.parentElement?.classList.contains("code-block-wrap")) return;

      const wrapper = document.createElement("div");
      wrapper.className = "code-block-wrap";
      pre.parentNode?.insertBefore(wrapper, pre);
      wrapper.appendChild(pre);

      const btn = document.createElement("button");
      btn.className = "code-copy-btn";
      btn.setAttribute("aria-label", "复制代码");
      btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>`;

      btn.addEventListener("click", () => {
        const code = pre.textContent || "";
        navigator.clipboard.writeText(code).then(() => {
          btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`;
          setTimeout(() => {
            btn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>`;
          }, 1500);
        });
      });

      wrapper.appendChild(btn);
    });

    return () => {
      document.querySelectorAll<HTMLElement>(".code-block-wrap").forEach((wrap) => {
        const pre = wrap.querySelector("pre");
        if (pre) {
          wrap.parentNode?.insertBefore(pre, wrap);
        }
        wrap.remove();
      });
    };
  }, []);

  return null;
}
