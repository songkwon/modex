"use client";

import { publicLogoURLs } from "@/lib/runtime-config";

export function BrandLogo({ alt }: { alt: string }) {
  const logos = publicLogoURLs();
  return (
    <span className="brand-logo-wrap" aria-hidden={alt ? undefined : true}>
      <img className="brand-logo brand-logo--light" src={logos.light} alt={alt} width={28} height={28} />
      <img className="brand-logo brand-logo--dark" src={logos.dark} alt="" width={28} height={28} />
    </span>
  );
}
