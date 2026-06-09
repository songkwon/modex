import "./globals.css";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Modex",
  description: "Module Documentation Experience"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>
        <div className="shell">
          <header className="topbar">
            <Link className="brand" href="/">Modex</Link>
            <nav className="nav">
              <Link href="/search">搜索</Link>
              <Link href="/me/recent">最近访问</Link>
              <Link href="/me/favorites">我的关注</Link>
              <Link href="/admin">管理</Link>
            </nav>
          </header>
          {children}
        </div>
      </body>
    </html>
  );
}
