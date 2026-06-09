import "./globals.css";
import type { Metadata, Viewport } from "next";
import Image from "next/image";
import Link from "next/link";
import { ThemeToggle } from "@/components/theme-toggle";

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
              <Link href="/search">搜索</Link>
              <Link href="/me/recent">最近访问</Link>
              <Link href="/me/favorites">我的关注</Link>
              <Link href="/admin">管理</Link>
              <ThemeToggle />
            </nav>
          </header>
          {children}
        </div>
      </body>
    </html>
  );
}
