import type { Metadata } from "next";
import Link from "next/link";
import { HelpPage } from "@/components/help-page";
import { site } from "@/lib/site";

const description =
  "Help with installing Burrow, connecting a local service, HTTPS tunnels, and webhook troubleshooting.";
export const metadata: Metadata = {
  title: "Support",
  description,
  alternates: { canonical: "/support" },
  openGraph: { title: "Support | Burrow", description, url: "/support" },
  twitter: { title: "Support | Burrow", description },
};
export default function SupportPage() {
  return (
    <HelpPage
      page="support"
      title="Support for Burrow."
      description={description}
    >
      <section>
        <h2>Start with your local app</h2>
        <p>
          Check that your app is running and reachable locally on the port you
          want to share. If it does not respond locally, fix that before opening
          a tunnel.
        </p>
        <p>
          Our <a href={site.clientDocs}>Burrow client documentation</a> covers
          installation and CLI usage. For your own server, see the{" "}
          <a href={site.serverDocs}>self-hosting documentation</a>.
        </p>
      </section>
      <section>
        <h2>A tunnel will not connect</h2>
        <p>
          Check your network connection, the selected server, and the error
          shown by the CLI. Include your operating system, Burrow version, and a
          redacted error if you email us. Never include owner credentials or
          authentication tokens.
        </p>
      </section>
      <section>
        <h2>A webhook or request fails</h2>
        <p>
          Confirm the method and path expected by your local app. Check its logs
          and the request information available in Burrow. Share only a
          sanitized example, not request bodies, authorization headers, or
          private tunnel URLs.
        </p>
      </section>
      <section>
        <h2>Share deliberately</h2>
        <p>
          A tunnel makes the selected local service reachable from the internet.
          Use your app&apos;s own authentication where needed and stop the
          tunnel when you finish.
        </p>
        <p>
          For feedback, privacy requests, or a private security report,{" "}
          <Link href="/contact">contact Burrow</Link>.
        </p>
      </section>
    </HelpPage>
  );
}
