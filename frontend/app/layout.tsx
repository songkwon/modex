import "./globals.css";
import type { Metadata, Viewport } from "next";
import Image from "next/image";
import Link from "next/link";
import { Search } from "lucide-react";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserMenu } from "@/components/user-menu";

const themeInit = `(function(){try{var t=localStorage.getItem('modex_theme')||'system';var d=t==='dark'||(t==='system'&&matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.dataset.theme=d?'dark':'light';}catch(e){}})();`;

export const metadata: Metadata = {
  title: "Modex",
  description: "Module Documentation Experience",
  manifest: "/manifest.webmanifest",
  icons: {
    icon: "/icon.svg",
    apple: "/apple-icon.png"
  }
};

export const viewport: Viewport = {
  themeColor: "#0E1F30"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInit }} />
      </head>
      <body>
        <div className="shell">
          <header className="topbar">
            <Link className="brand" href="/">
              <Image src="/logo.svg" alt="Modex" width={28} height={28} priority style={{ borderRadius: 8 }} />
              <span>Modex</span>
            </Link>
            <nav className="nav">
              <Link className="button icon-button" href="/search" title="搜索文档" aria-label="搜索文档"><Search size={16} /></Link>
              <ThemeToggle />
              <UserMenu />
            </nav>
          </header>
          {children}
        </div>
      </body>
    </html>
  );
}
