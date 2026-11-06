import { site } from "@/lib/site";
import Link from "next/link";

export function WorkflowStories() {
  return (
    <section
      className="page-container story-section"
      aria-labelledby="workflows-heading"
    >
      <div className="section-lead" data-reveal>
        <h2 id="workflows-heading" className="section-title">
          Multiple services and logs
        </h2>
      </div>
      <div className="workflow-pair" data-stagger>
        <article data-reveal>
          <div
            className="workflow-example workflow-ports"
            data-ui
            aria-label="Illustrative frontend and API ports"
          >
            <div>
              <span>Frontend</span>
              <code>localhost:3000</code>
            </div>
            <div>
              <span>API</span>
              <code>localhost:8080</code>
            </div>
          </div>
          <h3>Share more than one service</h3>
          <p>
            Give your frontend and API separate public links from the same
            client. The manager keeps the ports and URLs together.
          </p>
          <a href={site.clientDocs} className="text-link">
            Multi-port quickstart
          </a>
        </article>
        <article data-reveal>
          <div className="workflow-example" data-ui>
            <pre>
              <code>burrowctl logs &lt;id&gt; --follow</code>
            </pre>
          </div>
          <h3>Inspect requests</h3>
          <p>
            Watch incoming requests with follow and refresh controls. You can
            also inspect logs in a second terminal with the same owner
            credential.
          </p>
          <Link href="/docs/workflows" className="text-link">
            Webhook walkthrough
          </Link>
        </article>
      </div>
    </section>
  );
}
