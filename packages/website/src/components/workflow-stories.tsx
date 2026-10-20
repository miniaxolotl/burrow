import { ArrowUpRight, Code2, Link2, Webhook } from "lucide-react";

const stories = [
  {
    icon: Link2,
    number: "01",
    label: "THE QUICK LOOK",
    title: "“Can you take a look at this?”",
    text: "Give a teammate a working preview while the idea is still taking shape. Keep editing on your machine.",
    command: "burrowctl tunnel 3000",
    method: "GET",
    path: "/your-next-idea",
    result: "200 OK",
  },
  {
    icon: Webhook,
    number: "02",
    label: "THE REAL CALLBACK",
    title: "Let the webhook come to you.",
    text: "Point a provider’s test webhook at your HTTPS URL. Watch the request arrive in your local app.",
    command: "burrowctl tunnel 8080",
    method: "POST",
    path: "/webhooks/payment",
    result: "202 Accepted",
  },
  {
    icon: Code2,
    number: "03",
    label: "THE WHOLE PICTURE",
    title: "Frontend. API. One place.",
    text: "Keep multiple tunnels together in the terminal manager. Switch between them and inspect each request log.",
    command: "burrowctl",
    method: "GET",
    path: "/api/projects",
    result: "200 OK",
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
          <p className="eyebrow">MAKE ROOM FOR THE REAL WORLD</p>
          <h2 id="workflows-heading" className="section-heading mt-4">
            Good work deserves
            <br />
            <em>a way out.</em>
          </h2>
        </div>
        <p>
          From “does this look right?” to “did that callback arrive?”, a little
          tunnel goes a long way.
        </p>
      </div>
      <div className="workflow-grid">
        {stories.map(
          ({
            icon: Icon,
            number,
            label,
            title,
            text,
            command,
            method,
            path,
            result,
          }) => (
            <article className="workflow-story" key={number} data-reveal>
              <div className="workflow-heading">
                <Icon size={23} strokeWidth={1.4} aria-hidden="true" />
                <span>
                  {number} / {label}
                </span>
              </div>
              <h3>{title}</h3>
              <p>{text}</p>
              <div className="workflow-mini">
                <code>
                  <span>$</span> {command}
                </code>
                <div>
                  <b>{method}</b>
                  <span>{path}</span>
                </div>
                <small>
                  <span /> {result}{" "}
                  <span className="workflow-example">EXAMPLE</span>
                </small>
              </div>
              <a className="text-link" href="#install">
                Try it with Burrow <ArrowUpRight size={14} aria-hidden="true" />
              </a>
            </article>
          ),
        )}
      </div>
    </section>
  );
}
