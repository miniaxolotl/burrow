import { ArrowUpRight } from "lucide-react";
import Link from "next/link";
import { Brand } from "@/components/brand";
import { BrambleFamily } from "@/components/bramble-family";
import { ShareSite } from "@/components/share-site";
import { site } from "@/lib/site";
import { TelemetryChoice } from "@/components/website-telemetry";

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="page-container">
        <div className="footer-main">
          <div className="footer-brand">
            <Brand />
            <p>HTTPS tunnels for local development.</p>
          </div>
          <nav className="footer-links" aria-label="Footer navigation">
            <p className="eyebrow">EXPLORE</p>
            <Link
              href="/docs"
              data-analytics-event="docs_link_clicked"
              data-analytics-placement="footer"
            >
              Documentation
            </Link>
            <Link href="/#how-it-works">How it works</Link>
            <Link href="/#install">Install Burrow</Link>
          </nav>
          <div className="footer-links">
            <p className="eyebrow">STAY IN TOUCH</p>
            <Link href="/contact">
              Say hello <ArrowUpRight size={12} aria-hidden="true" />
            </Link>
            <a href={site.github}>
              GitHub <ArrowUpRight size={12} aria-hidden="true" />
            </a>
            <ShareSite url={site.url} />
          </div>
        </div>
        <BrambleFamily current="burrow" />
        <div className="footer-bottom">
          <span>Copyright {new Date().getFullYear()} Bramble Labs Inc.</span>
          <nav className="footer-utilities" aria-label="Legal and contact">
            <a href="/privacy">Privacy policy</a>
            <a href="/terms">Terms &amp; conditions</a>
            <TelemetryChoice />
            <Link href="/support">Support</Link>
            <Link href="/contact">Contact</Link>
            <a href={`${site.github}/blob/production/LICENSE`}>MIT license</a>
          </nav>
        </div>
      </div>
    </footer>
  );
}
