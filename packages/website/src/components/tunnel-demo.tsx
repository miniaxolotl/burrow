"use client";

import { ArrowRight, Check, RotateCcw } from "lucide-react";
import { useState } from "react";

const scenarios = [
  {
    label: "Share a preview",
    port: 3000,
    request: "GET /",
    status: "200 OK",
    slug: "softly-brave-little-fox",
    sender: "Teammate’s browser",
    destination: "Your frontend",
    description:
      "Your teammate opens the link. Their browser reaches the app running on your laptop.",
    detail:
      "Make a change locally. Refresh the same public URL to review it together.",
  },
  {
    label: "Test a webhook",
    port: 8080,
    request: "POST /webhooks/order",
    status: "202 Accepted",
    slug: "brightly-bold-blue-hare",
    sender: "Webhook provider",
    destination: "Your local handler",
    description:
      "Point a test webhook at your public URL. The incoming order event reaches your local handler.",
    detail:
      "Keep your debugger attached. Verify the response before switching the integration to production.",
  },
  {
    label: "Preview an API",
    port: 4000,
    request: "GET /api/projects",
    status: "200 OK",
    slug: "quietly-calm-green-owl",
    sender: "Teammate’s API client",
    destination: "Your local API",
    description:
      "Let a teammate try your API against the branch you are working on, without deploying it.",
    detail:
      "Keep authentication on sensitive endpoints. A public tunnel does not make a private API.",
  },
] as const;

export function TunnelDemo() {
  const [replay, setReplay] = useState(0);
  const [selected, setSelected] = useState(0);
  return (
    <figure
      className="tunnel-landscape"
      data-ui
      aria-label="A local development server connects through Burrow to a public HTTPS URL"
    >
      <div
        className="tunnel-scenarios"
        role="group"
        aria-label="Choose a tunnel example"
      >
        {scenarios.map((item, index) => (
          <button
            key={item.label}
            type="button"
            aria-pressed={selected === index}
            aria-controls="tunnel-example-output"
            onClick={() => {
              setSelected(index);
            }}
          >
            {item.label}
          </button>
        ))}
      </div>
      <div className="tunnel-terminal">
        <div className="terminal-titlebar">
          <span className="terminal-dots" aria-hidden="true">
            <i />
            <i />
            <i />
          </span>
          <span>burrowctl</span>
          <span>Interactive example</span>
        </div>
        <div
          className="tunnel-terminal-pages"
          key={replay}
          aria-live="polite"
          aria-atomic="true"
        >
          {scenarios.map((scenario, index) => (
            <div
              className="tunnel-terminal-body"
              id={selected === index ? "tunnel-example-output" : undefined}
              key={scenario.port}
              data-inactive={selected !== index}
              aria-hidden={selected !== index}
              inert={selected !== index}
            >
              <pre className="tunnel-command">
                <code>
                  <span className="terminal-prompt">$</span> burrowctl tunnel{" "}
                  {scenario.port}
                </code>
              </pre>
              <p className="tunnel-connected">
                <Check size={14} aria-hidden="true" /> Tunnel connected
              </p>
              <div className="tunnel-public-url">
                <span>Public HTTPS URL</span>
                <p>
                  <code>
                    https://{scenario.slug}
                    <strong>.brmbl.dev</strong>
                  </code>
                </p>
              </div>
              <div className="tunnel-request">
                <span>Local response</span>
                <span>{scenario.status}</span>
              </div>
              <ol className="request-route" aria-label="Incoming request path">
                <li>
                  <strong>{scenario.sender}</strong>
                  <code>{scenario.request}</code>
                </li>
                <li className="route-arrow" aria-hidden="true">
                  <ArrowRight size={24} />
                </li>
                <li>
                  <strong>Burrow</strong>
                  <span>Public HTTPS endpoint</span>
                </li>
                <li className="route-arrow" aria-hidden="true">
                  <ArrowRight size={24} />
                </li>
                <li>
                  <strong>{scenario.destination}</strong>
                  <code>localhost:{scenario.port}</code>
                </li>
              </ol>
            </div>
          ))}
        </div>
      </div>
      <div className="tunnel-scenario-story">
        {scenarios.map((scenario, index) => (
          <div
            key={scenario.port}
            className="tunnel-story-detail"
            data-inactive={selected !== index}
            aria-hidden={selected !== index}
            inert={selected !== index}
          >
            <p>{scenario.description}</p>
            <p>{scenario.detail}</p>
          </div>
        ))}
      </div>
      <figcaption>
        <span>Illustrated preview with example URL and request</span>
        <button
          className="tunnel-replay"
          type="button"
          onClick={() => setReplay((value) => value + 1)}
        >
          <RotateCcw size={12} aria-hidden="true" /> Replay example
        </button>
      </figcaption>
    </figure>
  );
}
