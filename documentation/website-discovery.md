# Public setup and reference discovery

The own-domain `/docs` hub publishes three explicitly selected documents:
`client.md`, `server.md` and `workflows.md`. These committed Markdown files are the
only reference bodies; the website reads them during static generation, not from
GitHub at request time. Update the original document, not a duplicated web copy.

`packages/website/src/content/documentation.ts` is the allowlisted source for paths,
metadata, navigation and sitemap coverage. Unknown slugs are 404; route inputs are
never filesystem paths. Missing source files fail builds. Preserve workspace files
outside the Vercel website root so the monorepo documentation is available at build.
No runtime Redis, tunnel server, directory listing or private endpoint is needed.

The pinned CommonMark/GFM renderer skips raw HTML and generates heading anchors.
Approved relative references become own-domain links; unpublished reference targets
remain explicit GitHub links. Keyboard-focusable code/table regions scroll locally
at narrow widths. These are server-rendered pages, not client-only Markdown viewers.

The hub explains the distinct npm dist-tag and GitHub stable/prerelease mechanisms
without changing tags, versions or the existing install recommendation. The owner
still needs to confirm the intended channel policy; an observed version mismatch
alone is not a defect. Do not confuse website-development Node requirements with
the CLI's actual runtime requirements.

## Verification

New browser regressions first reproduced 404 documents and the external-only docs
destination. They check no-JavaScript text, canonical/social URL metadata, sitemap
boundaries, heading fragments, cross-document links, invalid slugs, owner/public
boundaries in the worked examples and 320/390/768/1440px overflow. Localhost
builds intentionally keep an empty sitemap/noindex; an additional explicit-origin
production-like build verifies all four new public sitemap/canonical/200/index
responses without deploying. Existing consent,
telemetry, navigation, Go and release suites remain intact.

The webhook example is a disposable built-in Node service. Verify its POST/GET
behavior locally with synthetic data. A real public tunnel test is separate and
requires an explicitly approved target; source verification is not a production
forwarding, signature-validation or end-to-end provider acceptance claim.

See [reviewed captures](design/screenshots/discovery/README.md). No telemetry route
allowlist, robots/training preference, provider, registry, release or production
configuration is changed. No `llms.txt`/IndexNow/ranking/citation promise is made.
After a separately authorized release, verify actual source identity, public HTML,
links, sitemap and redirects before requesting appropriate engine updates. Use
owner-side Google/Bing evidence for index decisions rather than public fetch alone.
