import { connection } from "next/server";
import { LegalPage } from "@/components/legal-page";
import { privacy } from "@/content/legal";
import { pageMetadata } from "@/lib/page-metadata";

export const metadata = pageMetadata(
  "Privacy policy",
  privacy.summary,
  "/privacy",
);
export default async function PrivacyPage() {
  await connection();
  return <LegalPage document={privacy} />;
}
