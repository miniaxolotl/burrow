import "server-only";

const origin =
  process.env.SITE_URL ||
  (process.env.VERCEL_PROJECT_PRODUCTION_URL &&
    `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`) ||
  (process.env.RAILWAY_PUBLIC_DOMAIN &&
    `https://${process.env.RAILWAY_PUBLIC_DOMAIN}`) ||
  "http://localhost:3000";

const url = new URL(origin);

if (!["http:", "https:"].includes(url.protocol)) {
  throw new Error("SITE_URL must be an absolute HTTP or HTTPS URL.");
}

export const site = {
  name: "Burrow",
  title: "Burrow | Share localhost over HTTPS",
  creator: "Bramble Labs Inc.",
  description:
    "Give your local app a public HTTPS link. Share previews, test webhooks and inspect requests. Open source, with no account required. Install Burrow.",
  url: url.origin,
  indexable:
    process.env.SITE_NOINDEX !== "true" &&
    process.env.VERCEL_ENV !== "preview" &&
    !["localhost", "127.0.0.1", "[::1]"].includes(url.hostname),
  github: "https://github.com/miniaxolotl/burrow",
  npm: "https://www.npmjs.com/package/@miniaxolotl/burrowctl",
  releases: "https://github.com/miniaxolotl/burrow/releases",
  clientDocs:
    "https://github.com/miniaxolotl/burrow/blob/production/documentation/client.md",
  serverDocs:
    "https://github.com/miniaxolotl/burrow/blob/production/documentation/server.md",
} as const;
