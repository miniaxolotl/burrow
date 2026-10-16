import { Brand } from "@/components/brand";
import { ShareSite } from "@/components/share-site";
import { site } from "@/lib/site";

export function SiteFooter() {
  return (
    <footer className="page-container site-footer">
      <div className="footer-top">
        <div>
          <Brand />
          <p className="mt-3 text-xs text-muted">
            Made by <span className="font-medium text-ink">Bramble Labs</span>.
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
          <a
            href={`${site.github}/blob/production/LICENSE`}
            className="text-link"
          >
            MIT License
          </a>
        </nav>
      </div>
      <div className="footer-bottom">
        <span>Burrow · HTTPS tunnels for local development</span>
        <ShareSite url={site.url} />
      </div>
    </footer>
  );
}
