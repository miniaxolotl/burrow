"use client";

import { Check, Share2 } from "lucide-react";
import { useState } from "react";

export function ShareSite({ url }: { url: string }) {
  const [status, setStatus] = useState<"idle" | "copied" | "fallback">("idle");
  const [busy, setBusy] = useState(false);

  async function share() {
    setBusy(true);
    try {
      if (navigator.share) {
        try {
          await navigator.share({
            title: "Burrow | Share localhost over HTTPS",
            text: "A public HTTPS link for your local app. Open source, no account required.",
            url,
          });
          return;
        } catch (error) {
          if (error instanceof Error && error.name === "AbortError") return;
        }
      }
      await navigator.clipboard.writeText(url);
      setStatus("copied");
    } catch {
      setStatus("fallback");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <button
        type="button"
        onClick={share}
        disabled={busy}
        className="inline-flex min-h-11 items-center gap-2 text-xs font-medium underline-offset-4 hover:underline disabled:opacity-50"
      >
        {status === "copied" ? (
          <Check size={15} aria-hidden="true" />
        ) : (
          <Share2 size={15} aria-hidden="true" />
        )}
        {status === "copied" ? "Link copied" : "Share Burrow"}
      </button>
      <span role="status" className="sr-only">
        {status === "copied"
          ? "Burrow website link copied to your clipboard."
          : ""}
      </span>
      {status === "fallback" && (
        <label className="mt-2 block text-xs">
          Copy this link to share Burrow
          <input
            aria-label="Website link to share"
            readOnly
            value={url}
            onFocus={(event) => event.target.select()}
            className="mt-2 block w-full rounded-md border border-line bg-paper p-3 font-mono text-xs text-forest"
          />
        </label>
      )}
    </div>
  );
}
