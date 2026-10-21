import type { Metadata } from "next";
import { connection } from "next/server";
import { LegalPage } from "@/components/legal-page";
import { privacy } from "@/content/legal";

export const metadata: Metadata = {
  title: "Privacy policy",
  description: privacy.summary,
  alternates: { canonical: "/privacy" },
};
export default async function PrivacyPage() {
  await connection();
  return <LegalPage document={privacy} />;
}
