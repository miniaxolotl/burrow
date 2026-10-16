import {
  ArrowRight,
  ArrowUpRight,
  Check,
  ChevronDown,
  Layers3,
  Link2,
  Server,
  Webhook,
} from "lucide-react";
import { connection } from "next/server";
import { InstallCommands } from "@/components/install-commands";
import { ManagerPreview } from "@/components/manager-preview";
import { Navigation } from "@/components/navigation";
import { SiteFooter } from "@/components/site-footer";
import { StructuredData } from "@/components/structured-data";
import { TunnelDemo } from "@/components/tunnel-demo";
import { faqs } from "@/content/faq";
import { site } from "@/lib/site";

export default async function HomePage() {
  await connection();

  return (
    <>
      <Navigation github={site.github} />
      <main id="main">
        <section
          className="page-container hero-section"
          aria-labelledby="hero-heading"
        >
          <div className="min-w-0">
            <p className="eyebrow">BURROW / HTTPS TUNNELS</p>
            <h1 id="hero-heading" className="hero-heading">
              Localhost,
              <br />
              <em>shared.</em>
            </h1>
            <p className="hero-description">
              Give your local app a public HTTPS link. Share a preview, test
              webhooks, and keep building.
            </p>
            <div className="mt-7 flex flex-wrap items-center gap-x-6 gap-y-3">
              <a href="#install" className="button-primary">
                Install Burrow <ArrowRight size={16} aria-hidden="true" />
              </a>
              <a href={site.github} className="text-link">
                View source <ArrowUpRight size={14} aria-hidden="true" />
              </a>
            </div>
            <p className="mt-5 flex flex-wrap gap-x-4 gap-y-2 text-xs text-muted">
              <span className="inline-flex items-center gap-1.5">
                <Check size={12} aria-hidden="true" /> No account required
              </span>
              <span>Free & open source</span>
            </p>
          </div>
          <TunnelDemo />
        </section>

        <section
          id="features"
          className="page-container"
          aria-label="What you can do with Burrow"
        >
          <div className="feature-strip">
            {[
              {
                icon: Link2,
                title: "Share your work",
                description:
                  "Let a teammate try your local app without deploying it.",
              },
              {
                icon: Webhook,
                title: "Test real callbacks",
                description:
                  "Receive webhooks directly on your development server.",
              },
              {
                icon: Layers3,
                title: "Keep ports together",
                description:
                  "Manage your frontend, API, and request logs in one place.",
              },
            ].map(({ icon: Icon, title, description }) => (
              <article key={title}>
                <Icon
                  size={20}
                  strokeWidth={1.5}
                  className="mb-4 text-accent"
                  aria-hidden="true"
                />
                <h2 className="text-base font-medium tracking-tight">
                  {title}
                </h2>
                <p className="mt-2 text-sm leading-6 text-muted">
                  {description}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section
          id="how-it-works"
          className="dark-section section-space mt-16 sm:mt-20"
          aria-labelledby="manager-heading"
        >
          <div className="page-container">
            <div className="section-intro">
              <div>
                <p className="eyebrow text-highlight">THE TUNNEL MANAGER</p>
                <h2 id="manager-heading" className="section-heading mt-3">
                  One terminal.
                  <br />
                  <em>Your tunnels.</em>
                </h2>
              </div>
              <p>
                See your connection health, copy a URL, and inspect your
                requests. Try the preview below.
              </p>
            </div>
            <ManagerPreview />
          </div>
        </section>

        <section
          id="install"
          className="section-space bg-accent-soft/50"
          aria-labelledby="install-heading"
        >
          <div className="page-container grid items-center gap-9 lg:grid-cols-[0.9fr_1.1fr] lg:gap-16">
            <div>
              <p className="eyebrow">GET STARTED</p>
              <h2 id="install-heading" className="section-heading mt-3">
                Two commands.
                <br />
                <em>One public link.</em>
              </h2>
              <p className="mt-4 max-w-sm text-sm leading-7 text-muted">
                Install the CLI, point it at your app’s port, and share the URL.
                Works on macOS and Linux, on x64 and ARM64.
              </p>
              <a href={site.serverDocs} className="text-link mt-6">
                <Server size={15} aria-hidden="true" /> Prefer to self-host?{" "}
                <ArrowUpRight size={14} aria-hidden="true" />
              </a>
            </div>
            <InstallCommands releases={site.releases} />
          </div>
        </section>

        <section
          className="page-container section-space faq-section"
          aria-labelledby="faq-heading"
        >
          <div>
            <p className="eyebrow">GOOD TO KNOW</p>
            <h2 id="faq-heading" className="section-heading mt-3">
              A few details.
            </h2>
            <a href={site.clientDocs} className="text-link mt-4">
              Read the docs <ArrowUpRight size={14} aria-hidden="true" />
            </a>
          </div>
          <div className="border-t border-line">
            {faqs.map(({ question, answer }) => (
              <details key={question} name="faq" className="faq-item group">
                <summary>
                  {question}
                  <ChevronDown
                    size={16}
                    className="shrink-0 transition-transform group-open:rotate-180"
                    aria-hidden="true"
                  />
                </summary>
                <p>{answer}</p>
              </details>
            ))}
          </div>
        </section>
        <StructuredData />
      </main>
      <SiteFooter />
    </>
  );
}
