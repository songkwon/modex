import "./globals.css";
import type { Metadata, Viewport } from "next";
import Link from "next/link";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserMenu } from "@/components/user-menu";
import { SearchProvider } from "@/components/search-provider";
import { TopbarSearchButton } from "@/components/topbar-search";
import { TopbarChatButton } from "@/components/topbar-chat";
import { AnalyticsInit } from "@/components/analytics-init";
import { WelcomeGuideToast } from "@/components/welcome-guide-toast";
import { BrandLogo } from "@/components/brand-logo";
import { I18nProvider } from "@/lib/i18n";
import { getServerLocale } from "@/lib/i18n-server";
import { publicAppTitle, publicFaviconURL } from "@/lib/runtime-config";

const themeInit = `(function(){try{var t=localStorage.getItem('modex_theme')||'system';var d=t==='dark'||(t==='system'&&matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.dataset.theme=d?'dark':'light';}catch(e){}})();`;

export function generateMetadata(): Metadata {
  const appTitle = publicAppTitle();
  const faviconUrl = publicFaviconURL();
  return {
    title: appTitle,
    description: "Module Documentation Experience",
    manifest: "/manifest.webmanifest",
    icons: {
      icon: faviconUrl,
      apple: faviconUrl
    }
  };
}

export const viewport: Viewport = {
  themeColor: "#0E1F30"
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const locale = await getServerLocale();
  const appTitle = publicAppTitle();
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
                  <BrandLogo alt={appTitle} />
                  <span>{appTitle}</span>
                </Link>
                <nav className="nav">
                  <TopbarSearchButton />
                  <TopbarChatButton />
                  <ThemeToggle />
                  <UserMenu />
                </nav>
              </header>
              {children}
              <WelcomeGuideToast />
            </div>
          </SearchProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
