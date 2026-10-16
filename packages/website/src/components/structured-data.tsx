import { faqs } from "@/content/faq";
import { site } from "@/lib/site";

export function StructuredData() {
  const data = {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "WebSite",
        "@id": `${site.url}/#website`,
        name: site.name,
        url: site.url,
        description: site.description,
      },
      {
        "@type": "SoftwareApplication",
        "@id": `${site.url}/#application`,
        name: site.name,
        url: site.url,
        description: site.description,
        applicationCategory: "DeveloperApplication",
        operatingSystem: "macOS, Linux",
        downloadUrl: site.npm,
        license: `${site.github}/blob/production/LICENSE`,
        sameAs: [site.github, site.npm],
        author: { "@type": "Organization", name: site.creator },
        offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
      },
      {
        "@type": "FAQPage",
        mainEntity: faqs.map(({ question, answer }) => ({
          "@type": "Question",
          name: question,
          acceptedAnswer: { "@type": "Answer", text: answer },
        })),
      },
    ],
  };

  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{
        __html: JSON.stringify(data).replace(/</g, "\\u003c"),
      }}
    />
  );
}
