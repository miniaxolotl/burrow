import type { Metadata } from "next";
import { connection } from "next/server";
import { LegalPage } from "@/components/legal-page";
import { terms } from "@/content/legal";

export const metadata: Metadata = {
  title: "Terms and conditions | Burrow",
  description: terms.summary,
  alternates: { canonical: "/terms" },
};
export default async function TermsPage() {
  await connection();
  return <LegalPage document={terms} />;
}
