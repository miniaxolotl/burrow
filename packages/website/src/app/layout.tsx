import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
import { site } from "@/lib/site";
import {
  WebsiteSpeedInsights,
  WebsiteVercelAnalytics,
} from "@/components/website-speed-insights";
import { WebsiteTelemetry } from "@/components/website-telemetry";
import { WebsiteTagManager } from "@/components/website-tag-manager";
import { WebsiteCookieConsent } from "@/components/cookie-consent";
import "vanilla-cookieconsent/dist/cookieconsent.css";
import "./globals.css";
import "./brand-family.css";
import "./marketing-ui.css";
import "./site-chrome.css";
import "./cookie-consent.css";

const sans = localFont({
  src: [
    {
      path: "../../public/brand-kit/fonts/ibm-plex-sans-400.woff2",
      weight: "400",
    },
    {
      path: "../../public/brand-kit/fonts/ibm-plex-sans-500.woff2",
      weight: "500",
    },
    {
      path: "../../public/brand-kit/fonts/ibm-plex-sans-600.woff2",
      weight: "600",
    },
    {
      path: "../../public/brand-kit/fonts/ibm-plex-sans-700.woff2",
      weight: "700",
    },
  ],
  variable: "--font-plex-sans",
  display: "swap",
  adjustFontFallback: "Arial",
});

const mono = localFont({
  src: [
    {
      path: "../../public/brand-kit/fonts/ibm-plex-mono-400.woff2",
      weight: "400",
    },
    {
      path: "../../public/brand-kit/fonts/ibm-plex-mono-500.woff2",
      weight: "500",
    },
  ],
  variable: "--font-plex-mono",
  display: "swap",
  adjustFontFallback: false,
});

export const metadata: Metadata = {
  metadataBase: new URL(site.url),
  title: { default: site.title, template: "%s | Burrow" },
  description: site.description,
  applicationName: site.name,
  authors: [{ name: site.creator }],
  creator: site.creator,
  publisher: site.creator,
  category: "technology",
  alternates: { canonical: "/" },
  robots: { index: site.indexable, follow: site.indexable },
  openGraph: {
    type: "website",
    locale: "en_US",
    url: site.url,
    siteName: site.name,
    title: site.title,
    description: site.description,
  },
  twitter: {
    card: "summary_large_image",
    title: site.title,
    description: site.description,
  },
};

export const viewport: Viewport = {
  themeColor: "#f7f6f0",
  colorScheme: "light",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className={`${sans.variable} ${mono.variable}`}>
        <a href="#main" className="skip-link">
          Skip to content
        </a>
        {children}
        <WebsiteTelemetry />
        <WebsiteCookieConsent />
        <WebsiteTagManager />
        <WebsiteVercelAnalytics />
        {process.env.VERCEL_ENV === "production" && <WebsiteSpeedInsights />}
      </body>
    </html>
  );
}
