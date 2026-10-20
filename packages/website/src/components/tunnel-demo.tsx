import { ArrowRight, Check, Globe2, Terminal } from "lucide-react";
import { BurrowMark } from "@/components/burrow-mark";

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
        <div className="tunnel-identity">
          <BurrowMark size={144} />
        </div>
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
