import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Brand, BurrowMark } from "@/components/brand";

export const metadata: Metadata = {
  title: "A little off the beaten path",
  robots: { index: false, follow: false },
  alternates: { canonical: null },
};

export default function NotFound() {
  return (
    <div className="page-container py-8">
      <Brand />
      <main
        id="main"
        className="flex min-h-[75vh] flex-col items-center justify-center py-16 text-center"
      >
        <BurrowMark className="mb-8 size-20 text-forest/40" />
        <p className="eyebrow">404 / THIS TUNNEL ENDS HERE</p>
        <h1 className="section-heading mt-5">
          A little off
          <br />
          <em>the beaten path.</em>
        </h1>
        <p className="mt-6 text-sm text-muted">
          Let’s get you back to where the building happens.
        </p>
        <Link href="/" className="button-primary mt-8">
          <ArrowLeft size={16} aria-hidden="true" /> Back to Burrow
        </Link>
      </main>
    </div>
  );
}
