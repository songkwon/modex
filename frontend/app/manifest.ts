import type { MetadataRoute } from "next";
import { publicAppTitle, publicFaviconURL, publicLogoURL } from "@/lib/runtime-config";

function iconType(src: string): string | undefined {
  const lower = src.toLowerCase();
  if (lower.endsWith(".svg")) return "image/svg+xml";
  if (lower.endsWith(".png")) return "image/png";
  if (lower.endsWith(".webp")) return "image/webp";
  if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
  return undefined;
}

export default function manifest(): MetadataRoute.Manifest {
  const appTitle = publicAppTitle();
  const logoUrl = publicLogoURL();
  const faviconUrl = publicFaviconURL();
  const logoType = iconType(logoUrl);
  const faviconType = iconType(faviconUrl);
  return {
    name: appTitle,
    short_name: appTitle,
    description: "Module Documentation Experience",
    start_url: "/",
    display: "standalone",
    background_color: "#0E1F30",
    theme_color: "#0E1F30",
    icons: [
      { src: faviconUrl, sizes: "any", type: faviconType },
      { src: logoUrl, sizes: "any", type: logoType },
      { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" }
    ]
  };
}
