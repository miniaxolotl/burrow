import type { Metadata } from "next";
import Link from "next/link";
import { HelpPage } from "@/components/help-page";

const description =
  "Contact Burrow for help, feedback, privacy requests, or a private security report.";
export const metadata: Metadata = {
  title: "Contact",
  description,
  alternates: { canonical: "/contact" },
  openGraph: { title: "Contact | Burrow", description, url: "/contact" },
  twitter: { title: "Contact | Burrow", description },
};
export default function ContactPage() {
  return (
    <HelpPage
      page="contact"
      title="Contact Burrow."
      description="Have a question, found a problem, or want to suggest an improvement? Email us."
    >
      <section>
        <h2>Questions and feedback</h2>
        <p>
          Tell us what you are trying to do and where you got stuck. The{" "}
          <Link href="/support">support page</Link> has troubleshooting steps
          and links to the client and self-hosting documentation.
        </p>
      </section>
      <section>
        <h2>Reporting a problem</h2>
        <ul>
          <li>Your operating system and Burrow version.</li>
          <li>The command or action, with credentials removed.</li>
          <li>What you expected and what happened instead.</li>
          <li>A redacted error and the approximate time it occurred.</li>
        </ul>
        <p>
          Do not send owner credentials, tokens, private request bodies, or
          sensitive live tunnel URLs.
        </p>
      </section>
      <section>
        <h2>Security and privacy</h2>
        <p>
          Use the same email address to report a suspected vulnerability
          privately or ask about your information. Start with a brief
          description without exploit secrets or personal data. We can discuss
          what is needed next.
        </p>
        <p>
          Our <Link href="/privacy">privacy policy</Link> explains how Burrow
          handles service and support information.
        </p>
      </section>
    </HelpPage>
  );
}
