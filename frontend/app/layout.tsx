import "./globals.css";
import type { Metadata, Viewport } from "next";
import Image from "next/image";
import Link from "next/link";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserMenu } from "@/components/user-menu";
import { SearchProvider } from "@/components/search-provider";
import { TopbarSearchButton } from "@/components/topbar-search";
import { TopbarChatButton } from "@/components/topbar-chat";
import { AnalyticsInit } from "@/components/analytics-init";
import { I18nProvider } from "@/lib/i18n";
import { getServerLocale } from "@/lib/i18n-server";

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

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const locale = await getServerLocale();
  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <script src="/runtime-env.js" />
        <script dangerouslySetInnerHTML={{ __html: themeInit }} />
      </head>
      <body>
        <I18nProvider initialLocale={locale}>
          <AnalyticsInit />
          <SearchProvider>
            <div className="shell">
              <header className="topbar">
                <Link className="brand" href="/">
                  <Image src="/logo.svg" alt="Modex" width={28} height={28} style={{ borderRadius: 8 }} />
                  <span>Modex</span>
                </Link>
                <nav className="nav">
                  <TopbarSearchButton />
                  <TopbarChatButton />
                  <ThemeToggle />
                  <UserMenu />
                </nav>
              </header>
              {children}
            </div>
          </SearchProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
