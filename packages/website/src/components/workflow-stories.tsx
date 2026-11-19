"use client";

import { useId, useState } from "react";
import Link from "next/link";
import styles from "./workflow-stories.module.css";

const services = [
  {
    name: "Frontend",
    port: 3000,
    id: "softly-brave-little-fox",
    requests: [
      {
        time: "14:32:12",
        method: "GET",
        path: "/",
        status: 200,
        duration: "12 ms",
      },
      {
        time: "14:32:11",
        method: "GET",
        path: "/assets/app.js",
        status: 200,
        duration: "18 ms",
      },
      {
        time: "14:32:10",
        method: "GET",
        path: "/favicon.svg",
        status: 200,
        duration: "8 ms",
      },
    ],
    note: "This link serves the page and its assets. Keep the local app and Burrow running.",
  },
  {
    name: "API",
    port: 8080,
    id: "softly-calm-blue-owl",
    requests: [
      {
        time: "14:32:12",
        method: "POST",
        path: "/webhooks/order",
        status: 202,
        duration: "24 ms",
      },
      {
        time: "14:32:10",
        method: "GET",
        path: "/api/orders",
        status: 404,
        duration: "6 ms",
      },
      {
        time: "14:32:08",
        method: "GET",
        path: "/health",
        status: 200,
        duration: "4 ms",
      },
    ],
    note: "404: no matching route. 202: request accepted, not proof an order was processed.",
  },
] as const;

export function WorkflowStories() {
  const [selected, setSelected] = useState(0);
  const exampleId = useId();

  return (
    <section
      className="page-container story-section"
      aria-labelledby="workflows-heading"
    >
      <div className="section-lead" data-reveal>
        <h2 id="workflows-heading" className="section-title">
          Multiple services and logs
        </h2>
        <p>
          Give each local service its own public link. Keep its requests with
          the port they reached.
        </p>
      </div>
      <figure
        className={styles.example}
        data-ui
        aria-label="Example services and their request logs"
        aria-describedby={`${exampleId}-caption`}
        data-reveal
      >
        <div className={styles.command}>
          <span>Connect both services</span>
          <code>burrowctl tunnel 3000 8080</code>
        </div>
        <div className={styles.workspace}>
          <div className={styles.services}>
            <h3>Local services</h3>
            <div
              role="group"
              aria-label="Select an example service"
              className={styles.choices}
            >
              {services.map((service, index) => (
                <button
                  key={service.id}
                  type="button"
                  aria-label={`${service.name} Port ${service.port}: view example logs`}
                  aria-pressed={selected === index}
                  aria-controls={`${exampleId}-${service.port}`}
                  onClick={() => setSelected(index)}
                >
                  <strong>{service.name}</strong>
                  <span>Port {service.port}</span>
                </button>
              ))}
            </div>
          </div>
          <p className="sr-only" role="status">
            Showing {services[selected].name} example requests on port{" "}
            {services[selected].port}.
          </p>
          <div className={styles.panels}>
            {services.map((service, index) => (
              <div
                key={service.id}
                id={`${exampleId}-${service.port}`}
                className={styles.panel}
                data-inactive={selected !== index}
                aria-hidden={selected !== index}
                inert={selected !== index}
                role="region"
                aria-label={`${service.name} example request logs`}
              >
                <div className={styles.endpoint}>
                  <h3>{service.name} public link</h3>
                  <code>https://{service.id}.example.com</code>
                  <p>
                    Forwards to <code>localhost:{service.port}</code>
                  </p>
                </div>
                <table>
                  <caption>
                    Recent requests <span>· newest first</span>
                  </caption>
                  <thead>
                    <tr>
                      <th scope="col">Request</th>
                      <th scope="col">Status</th>
                      <th scope="col">Duration</th>
                    </tr>
                  </thead>
                  <tbody>
                    {service.requests.map((request) => (
                      <tr key={`${request.time}-${request.path}`}>
                        <td>
                          <div className={styles.requestMeta}>
                            <code className={styles.request}>
                              {request.method} {request.path}
                            </code>
                            <time>{request.time}</time>
                          </div>
                        </td>
                        <td>
                          <span
                            className={styles.status}
                            data-error={request.status >= 400}
                          >
                            {request.status}
                          </span>
                        </td>
                        <td>{request.duration}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <p className={styles.note}>{service.note}</p>
                <div className={styles.follow}>
                  <span>Follow in another terminal</span>
                  <code>burrowctl logs {service.id} --follow</code>
                </div>
              </div>
            ))}
          </div>
        </div>
        <figcaption id={`${exampleId}-caption`}>
          Local-only example. Fictional links and traffic; no tunnel is opened.
        </figcaption>
      </figure>
      <div className={styles.links}>
        <Link
          href="/docs/workflows#share-two-local-services"
          className="text-link"
        >
          Multi-service guide
        </Link>
        <Link
          href="/docs/workflows#inspect-only-your-tunnel"
          className="text-link"
        >
          Inspect your own requests
        </Link>
      </div>
    </section>
  );
}
