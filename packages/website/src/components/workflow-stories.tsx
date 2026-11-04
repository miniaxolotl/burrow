import { site } from "@/lib/site";
import Link from "next/link";
import Image from "next/image";

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
          <div className="workflow-photo">
            <Image
              src="/images/kit/burrow-code.webp"
              alt="Application code on a laptop"
              fill
              sizes="(max-width: 767px) calc(100vw - 40px), 50vw"
            />
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
