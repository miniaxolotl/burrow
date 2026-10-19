import { ArrowRight, Check, Globe2, Terminal } from "lucide-react";

export function TunnelDemo() {
  return (
    <figure
      className="tunnel-landscape"
      aria-label="A local development server connects through Burrow to a public HTTPS URL"
    >
      <div className="tunnel-scene">
        <div className="tunnel-scene-label">
          <span>SMALL TUNNEL. BIG POSSIBILITIES.</span>
          <span aria-hidden="true">↗</span>
        </div>
        <svg
          className="tunnel-portal"
          viewBox="0 -20 500 280"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M75 260V180a175 175 0 0 1 350 0v80"
            stroke="#dcb399"
            strokeWidth="34"
          />
          <path
            d="M121 260V180a129 129 0 0 1 258 0v80"
            stroke="#bf8060"
            strokeWidth="28"
          />
          <path
            d="M163 260V180a87 87 0 0 1 174 0v80"
            stroke="#9d4930"
            strokeWidth="23"
          />
          <path
            d="M205 260V180a45 45 0 0 1 90 0v80"
            stroke="#253e31"
            strokeWidth="19"
          />
          <path
            className="tunnel-packet"
            d="M121 260V180a129 129 0 0 1 258 0v80"
            stroke="#fff8e9"
            strokeWidth="5"
            strokeLinecap="round"
            pathLength="100"
          />
        </svg>
        <div className="tunnel-endpoint endpoint-local">
          <Terminal size={17} aria-hidden="true" />
          <span>
            Your machine<strong>localhost:3000</strong>
          </span>
        </div>
        <div className="tunnel-endpoint endpoint-world">
          <Globe2 size={17} aria-hidden="true" />
          <span>
            The internet<strong>Hello, world.</strong>
          </span>
        </div>
        <span className="tunnel-seal">
          LOCAL
          <br />
          <ArrowRight size={18} aria-hidden="true" />
          <br />
          LIVE
        </span>
      </div>
      <div className="tunnel-terminal">
        <div className="terminal-titlebar">
          <span className="terminal-dots" aria-hidden="true">
            <i />
            <i />
            <i />
          </span>
          <span>burrowctl</span>
          <span>EXAMPLE</span>
        </div>
        <div className="tunnel-terminal-body">
          <p>
            <span className="terminal-prompt">$</span> burrowctl tunnel 3000
          </p>
          <p className="tunnel-connected">
            <Check size={14} aria-hidden="true" /> Tunnel connected
          </p>
          <div className="tunnel-public-url">
            <span>PUBLIC HTTPS URL</span>
            <p>
              https://softly-brave-little-fox<strong>.brmbl.dev</strong>
            </p>
          </div>
          <div className="tunnel-request">
            <span>GET /</span>
            <span>200 OK</span>
            <span>→ localhost:3000</span>
          </div>
        </div>
      </div>
      <figcaption>
        Keep coding locally. Send the link. See what comes through.
        <br />
        <span>Illustrated preview · example URL and request</span>
      </figcaption>
    </figure>
  );
}
