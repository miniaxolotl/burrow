import { site } from "@/lib/site";

export const dynamic = "force-static";

export function GET() {
  return new Response(
    `# Burrow

> ${site.description}

Burrow is free and open source for macOS and Linux, by Bramble Labs Inc. No account is required. A public tunnel URL is not an access-control boundary: protect sensitive applications yourself. A tunnel forwards to an already-running local service; it does not deploy that service.

## Quick start
Install: npm install -g @miniaxolotl/burrowctl
Run your local app, then: burrowctl tunnel 3000
Replace 3000 with the app's port and keep the client running.

## Public references
- [Burrow](${site.url}/): product overview and installation.
- [Documentation](${site.url}/docs): public reference hub.
- [Client](${site.url}/docs/client): installation and CLI reference.
- [Workflows](${site.url}/docs/workflows): previews and webhook examples.
- [Server](${site.url}/docs/server): self-hosted service reference.
- [Deploy](${site.url}/docs/deploy): deployment, release policy and load testing.
- [Source](${site.github}): code and license.
- [Support](${site.url}/support): product help.
- [Contact](${site.url}/contact): public contact details.
- [Privacy](${site.url}/privacy): privacy and telemetry practices.
- [Terms](${site.url}/terms): terms of use.

Website terminal examples contain sample URLs and requests, not live traffic. This guide describes public documentation, not private tunnel management or user traffic.
`,
    {
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
        "X-Content-Type-Options": "nosniff",
      },
    },
  );
}
