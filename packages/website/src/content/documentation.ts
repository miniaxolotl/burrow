// Public reference allowlist: never enumerate arbitrary repository files.
export const documentation = [
  {
    slug: "client",
    source: "client.md",
    title: "burrowctl",
    description:
      "Install the Burrow client, share local services, manage your tunnels and understand owner-only access.",
  },
  {
    slug: "workflows",
    source: "workflows.md",
    title: "Webhooks and multiple local services",
    description:
      "Try a disposable webhook endpoint, share multiple ports, inspect your own request logs and clean up safely.",
  },
  {
    slug: "server",
    source: "server.md",
    title: "burrowd",
    description:
      "Run the Burrow server with Redis and TLS, and understand management authentication and ownership retention.",
  },
] as const;

export function documentationHref(href: string, source: string): string {
  if (/^(?:[a-z][a-z\d+.-]*:|\/|#)/i.test(href)) return href;
  const resolved = new URL(
    href,
    `https://reference.invalid/documentation/${source}`,
  );
  const document = documentation.find(
    (item) => resolved.pathname === `/documentation/${item.source}`,
  );
  return document
    ? `/docs/${document.slug}${resolved.search}${resolved.hash}`
    : `https://github.com/miniaxolotl/burrow/blob/production${resolved.pathname}${resolved.search}${resolved.hash}`;
}
