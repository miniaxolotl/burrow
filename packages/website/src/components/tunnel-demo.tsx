import { ArrowRight, Check, Globe2, Terminal } from "lucide-react";

export function TunnelDemo() {
  return (
    <figure className="product-preview">
      <div className="mb-5 flex items-center justify-between gap-3 text-xs text-muted">
        <span className="flex items-center gap-2">
          <Terminal size={14} aria-hidden="true" /> Your machine
        </span>
        <ArrowRight size={14} aria-hidden="true" />
        <span className="flex items-center gap-2">
          <Globe2 size={14} aria-hidden="true" /> The internet
        </span>
      </div>
      <div className="overflow-hidden rounded-lg border border-ink/10 bg-terminal text-paper">
        <div className="terminal-titlebar">
          <span className="flex gap-1.5" aria-hidden="true">
            <i className="size-2 rounded-full bg-[#c18b7b]" />
            <i className="size-2 rounded-full bg-[#c3b777]" />
            <i className="size-2 rounded-full bg-[#82a682]" />
          </span>
          <span>burrowctl</span>
          <span className="text-[9px] tracking-wider">DEMO</span>
        </div>
        <div className="p-5 font-mono text-[11px] leading-6 sm:p-6">
          <p className="break-words">
            <span className="mr-2 text-highlight">$</span>burrowctl tunnel
            create
            <br />
            <span className="pl-4 text-terminal-muted">--port 3000</span>
          </p>
          <p className="mt-6 flex items-center gap-2 text-highlight">
            <Check size={14} aria-hidden="true" /> Tunnel connected
          </p>
          <div className="mt-4 border-t border-white/10 pt-4">
            <p className="text-[10px] text-terminal-muted">PUBLIC HTTPS URL</p>
            <p className="mt-2 break-all text-paper">
              https://softly-brave-little-fox
              <span className="text-highlight">.burrow.mawa.dev</span>
            </p>
            <p className="mt-3 flex items-center gap-2 text-terminal-muted">
              <ArrowRight size={12} aria-hidden="true" /> localhost:3000
            </p>
          </div>
        </div>
      </div>
      <figcaption className="preview-caption">
        Illustrated preview · example URL
      </figcaption>
    </figure>
  );
}
