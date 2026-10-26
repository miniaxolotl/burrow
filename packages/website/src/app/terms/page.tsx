import { connection } from "next/server";
import { LegalPage } from "@/components/legal-page";
import { terms } from "@/content/legal";
import { pageMetadata } from "@/lib/page-metadata";

export const metadata = pageMetadata(
  "Terms and conditions",
  terms.summary,
  "/terms",
);
export default async function TermsPage() {
  await connection();
  return <LegalPage document={terms} />;
}
