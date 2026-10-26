import Link from "next/link";
import { documentation } from "@/content/documentation";
import { DocumentationPage } from "@/components/documentation-page";
import { pageMetadata } from "@/lib/page-metadata";
import { site } from "@/lib/site";

const description =
  "Install Burrow, share local services over HTTPS, test webhooks and understand the client and self-hosted server.";
export const metadata = pageMetadata(
  "Burrow documentation",
  description,
  "/docs",
);
export default function DocsPage() {
  return (
    <DocumentationPage title="Burrow documentation" description={description}>
      <h2>Choose a reference</h2>
      <dl>
        {documentation.map((item) => (
          <div key={item.slug} className="mb-6">
            <dt>
              <Link href={`/docs/${item.slug}`}>{item.title}</Link>
            </dt>
            <dd>{item.description}</dd>
          </div>
        ))}
      </dl>
      <h2>Check the installation channel</h2>
      <p>
        The npm install command follows the package&apos;s{" "}
        <a href={site.npm}>latest dist-tag</a>. GitHub separately labels
        releases as stable or prerelease; its latest-release shortcut omits
        prereleases. These labels need not select the same version.
      </p>
      <p>
        Review the actual version and release notes for your chosen channel
        before installing. These references are built from this website&apos;s
        committed source; an older installed binary may differ. This
        documentation does not relabel published releases or change the
        recommended install command.
      </p>
      <h2>Before exposing an application</h2>
      <p>
        A tunnel makes the application reachable from the internet. Owner
        credentials protect tunnel management, not the application&apos;s own
        routes. Use appropriate application authentication and synthetic data
        for examples.
      </p>
      <p>
        <Link href="/support">Get help</Link> with a setup problem. Keep tokens
        and private request payloads out of support messages.
      </p>
    </DocumentationPage>
  );
}
