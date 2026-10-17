import { Brand } from "@/components/brand";
import { BrambleFamily } from "@/components/bramble-family";
import { ShareSite } from "@/components/share-site";
import { site } from "@/lib/site";

export function SiteFooter() {
  return (
    <footer className="page-container site-footer">
      <div className="footer-top">
        <div>
          <Brand />
          <p className="mt-3 text-xs leading-6 text-muted">
            HTTPS tunnels for local development.
            <br />A developer tool by Bramble Labs Inc.
          </p>
        </div>
        <nav
          aria-label="Footer navigation"
          className="flex flex-wrap gap-x-6 gap-y-2"
        >
          <a href={site.clientDocs} className="text-link">
            Documentation
          </a>
          <a href={site.github} className="text-link">
            GitHub
          </a>
          <a href="https://bramblelabs.co/privacy" className="text-link">
            Website privacy
          </a>
          <a
            href={`${site.github}/blob/production/LICENSE`}
            className="text-link"
          >
            MIT License
          </a>
        </nav>
      </div>
      <BrambleFamily current="burrow" />
      <div className="footer-bottom">
        <span>© {new Date().getFullYear()} Bramble Labs Inc.</span>
        <ShareSite url={site.url} />
      </div>
    </footer>
  );
}
