import { ArrowRight, ChevronDown } from "lucide-react";
import { connection } from "next/server";
import { InstallCommands } from "@/components/install-commands";
import { ManagerPreview } from "@/components/manager-preview";
import { Navigation } from "@/components/navigation";
import { SiteFooter } from "@/components/site-footer";
import { StructuredData } from "@/components/structured-data";
import { TunnelDemo } from "@/components/tunnel-demo";
import { WorkflowStories } from "@/components/workflow-stories";
import { MarketingMotion } from "@/components/marketing-motion";
import { faqs } from "@/content/faq";
import { site } from "@/lib/site";
import { HeroCtaCopy } from "@/components/website-telemetry";

const features = [
  [
    "Share more than one service",
    "Give your frontend and API separate public links from one client with burrowctl tunnel 3000 8080. The manager keeps the ports and URLs together.",
  ],
  [
    "Inspect only your requests",
    "List your tunnels, filter logs by method, status, path or duration, and follow new requests live. History stays owner-scoped and never leaves your machine without you.",
  ],
  [
    "Reconnect without a new link",
    "While the CLI is running, transient connection failures retry the same URL. Explicitly closing a tunnel revokes it; a new tunnel gets a fresh URL.",
  ],
  [
    "Owner-only controls",
    "The client creates a private owner credential automatically. Sharing a public URL doesn't grant someone access to list, inspect or close your tunnels.",
  ],
  [
    "No account needed",
    "There is no signup, login or manual credential setup. Each server gets its own local owner identity, and a shareable link never carries management access.",
  ],
  [
    "Self-hosting",
    "Run the open-source burrowd server on your own infrastructure and domain. Point the CLI at it with the --server option.",
  ],
] as const;

export default async function HomePage() {
  await connection();
  return (
    <MarketingMotion>
      <Navigation github={site.github} />
      <main id="main" className="product-redesign terminal-home">
        <div className="hero-field">
          <section
            className="page-container product-hero hero-section"
            aria-labelledby="hero-heading"
          >
            <div className="product-hero-copy">
              <h1 id="hero-heading" className="hero-title">
                Your local app.
                <br />A public link.
              </h1>
              <p className="hero-summary">
                Share what’s running on your machine. Get feedback, test
                webhooks and inspect requests without deploying.
              </p>
              <div className="hero-actions">
                <a
                  href="#install"
                  className="button-primary"
                  data-analytics-event="cta_clicked"
                  data-analytics-placement="hero"
                >
                  <HeroCtaCopy />
                  <ArrowRight size={17} aria-hidden="true" />
                </a>
                <a href={site.clientDocs} className="text-link">
                  Read the docs
                </a>
              </div>
              <p className="hero-proof">
                Free and open source. macOS and Linux. No account needed.
              </p>
            </div>
            <div className="product-stage terminal-stage" data-reveal>
              <TunnelDemo />
            </div>
          </section>
        </div>
        <section
          id="install"
          className="feature-index"
          aria-labelledby="install-heading"
        >
          <div className="page-container install-layout">
            <div data-reveal>
              <h2 id="install-heading" className="section-title">
                Install and connect
              </h2>
              <p>
                Install the CLI, run your local app, then open a tunnel. Use npm
                or a native binary for macOS or Linux, on x64 or ARM64.
              </p>
              <a href={site.serverDocs} className="text-link">
                Prefer to self-host?
              </a>
            </div>
            <InstallCommands releases={site.releases} />
          </div>
        </section>
        <WorkflowStories />
        <section
          id="features"
          className="feature-index"
          aria-labelledby="features-heading"
        >
          <div className="page-container feature-index-layout">
            <div className="feature-index-heading" data-reveal>
              <h2 id="features-heading" className="section-title">
                How tunnels work
              </h2>
              <a href={site.github} className="text-link">
                Explore the source
              </a>
            </div>
            <dl className="feature-list" data-stagger>
              {features.map(([title, description]) => (
                <div className="feature-row" key={title} data-reveal>
                  <dt>{title}</dt>
                  <dd>{description}</dd>
                </div>
              ))}
            </dl>
          </div>
        </section>
        <section
          id="how-it-works"
          className="page-container story-section"
          aria-labelledby="manager-heading"
        >
          <div className="section-lead" data-reveal>
            <h2 id="manager-heading" className="section-title">
              Tunnel manager
            </h2>
            <p>
              Add ports, copy links and inspect requests from one terminal. Try
              the illustrated manager below with sample tunnels. Prefer scripts?
              The same CLI runs headless with JSON output, and{" "}
              <code>burrowctl doctor</code> checks your setup without opening a
              tunnel.
            </p>
          </div>
          <ManagerPreview />
        </section>
        <section
          className="page-container product-facts"
          aria-labelledby="public-heading"
          data-reveal
        >
          <h2 id="public-heading" className="section-title">
            Public access and privacy
          </h2>
          <div>
            <p>
              A tunnel URL makes your local service reachable from the internet.
              Owner-only management protects tunnel controls, not the app behind
              the link. Add authentication to your app before exposing sensitive
              data.
            </p>
            <p>
              Official CLI releases collect anonymous usage outcomes and
              sanitized diagnostics. Turn them off with{" "}
              <code>burrowctl telemetry disable</code>.{" "}
              <a href={site.clientDocs}>Read the client guide</a> for ownership,
              reconnection and privacy details.
            </p>
          </div>
        </section>
        <section
          className="page-container story-section faq-section"
          aria-labelledby="faq-heading"
        >
          <div>
            <h2 id="faq-heading" className="section-title">
              Questions
            </h2>
            <a href="/support" className="text-link">
              Get support
            </a>
          </div>
          <div className="question-list">
            {faqs.map(({ question, answer }) => (
              <details key={question} name="faq" className="faq-item">
                <summary>
                  {question}
                  <ChevronDown size={18} aria-hidden="true" />
                </summary>
                <p>{answer}</p>
              </details>
            ))}
          </div>
        </section>
        <StructuredData />
      </main>
      <SiteFooter />
    </MarketingMotion>
  );
}
