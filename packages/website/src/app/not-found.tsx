import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Brand, BurrowMark } from "@/components/brand";

export const metadata: Metadata = {
  title: "Page not found",
  robots: { index: false, follow: false },
  alternates: { canonical: null },
};

export default function NotFound() {
  return (
    <div className="page-container py-8">
      <Brand />
      <main
        id="main"
        className="recovery-page flex min-h-[75vh] flex-col items-center justify-center py-16 text-center"
      >
        <span className="recovery-mark">
          <BurrowMark className="size-14 text-accent" />
        </span>
        <p className="eyebrow">404</p>
        <h1 className="section-heading mt-5">
          Page <em>not found.</em>
        </h1>
        <p className="mt-6 text-sm text-muted">
          We could not find that page. Return to the homepage.
        </p>
        <Link href="/" className="button-primary mt-8">
          <ArrowLeft size={16} aria-hidden="true" /> Back to Burrow
        </Link>
        <nav className="recovery-links" aria-label="Helpful links">
          <Link href="/#install">Installation</Link>
          <Link href="/support">Get help</Link>
        </nav>
      </main>
    </div>
  );
}
