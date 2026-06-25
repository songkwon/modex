import type { MetadataRoute } from "next";
import { publicAppTitle } from "@/lib/runtime-config";

export default function manifest(): MetadataRoute.Manifest {
  const appTitle = publicAppTitle();
  return {
    name: appTitle,
    short_name: appTitle,
    description: "Module Documentation Experience",
    start_url: "/",
    display: "standalone",
    background_color: "#0E1F30",
    theme_color: "#0E1F30",
    icons: [
      { src: "/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" }
    ]
  };
}
