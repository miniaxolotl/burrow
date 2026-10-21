import Image from "next/image";
import { ArrowUpRight } from "lucide-react";
import { site } from "@/lib/site";

const stories = [
  {
    image: "tunnel",
    title: "Get feedback sooner.",
    text: "Send a teammate a link to your local app. Keep editing while they try it.",
    alt: "Example workflow connecting localhost port 3000 to a public HTTPS link.",
    href: "#install",
    action: "Share a local app",
  },
  {
    image: "webhooks",
    title: "Test the real webhook.",
    text: "Point a test webhook at your tunnel. See the request reach your local server.",
    alt: "Example POST request to a webhook forwarded to localhost with a 200 response.",
    href: "#install",
    action: "Test a webhook",
  },
  {
    image: "self-host",
    title: "Run it on your terms.",
    text: "Host the open-source server on your own infrastructure and domain.",
    alt: "A local app connects through burrowd to a public HTTPS endpoint.",
    href: site.serverDocs,
    action: "Read the server guide",
  },
];

export function WorkflowStories() {
  return (
    <section
      className="page-container section-space"
      aria-labelledby="workflows-heading"
    >
      <div className="section-intro" data-reveal>
        <div>
          <p className="eyebrow">FROM LOCALHOST TO A WORKING LINK</p>
          <h2 id="workflows-heading" className="section-heading mt-4">
            Share it.
            <br />
            <em>Test it. Keep building.</em>
          </h2>
        </div>
        <p>Your app stays on your machine. Burrow handles the public link.</p>
      </div>
      <div className="workflow-grid">
        {stories.map(({ image, title, text, alt, href, action }, index) => (
          <article className="workflow-story" key={image} data-reveal>
            <div className="workflow-heading">
              <span>0{index + 1}</span>
            </div>
            <h3>{title}</h3>
            <p>{text}</p>
            <Image
              className="campaign-illustration"
              src={`/materials/${image}.svg`}
              alt={alt}
              width={900}
              height={520}
              sizes="(max-width: 767px) 100vw, 33vw"
            />
            <a className="text-link" href={href}>
              {action} <ArrowUpRight size={14} aria-hidden="true" />
            </a>
          </article>
        ))}
      </div>
      <p className="mt-5 text-xs text-muted">
        Illustrated workflows with example requests.
      </p>
    </section>
  );
}
