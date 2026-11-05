"use client";

import { Check, Share2 } from "lucide-react";
import { useId, useRef, useState } from "react";
import styles from "./share-site.module.css";

export function ShareSite({ url }: { url: string }) {
  const [status, setStatus] = useState<"idle" | "copied" | "fallback">("idle");
  const [busy, setBusy] = useState(false);
  const fallback = useRef<HTMLDialogElement>(null);
  const heading = useId();

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
      fallback.current?.showModal();
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
        <span className={styles.label}>
          <span aria-hidden={status === "copied"}>Share Burrow</span>
          <span aria-hidden={status !== "copied"}>Link copied</span>
        </span>
      </button>
      <span role="status" className="sr-only">
        {status === "copied"
          ? "Burrow website link copied to your clipboard."
          : ""}
      </span>
      <dialog
        ref={fallback}
        aria-labelledby={heading}
        className={styles.dialog}
      >
        <form method="dialog">
          <h2 id={heading}>Share Burrow</h2>
          <label>
            Copy this link to share Burrow
            <input
              aria-label="Website link to share"
              readOnly
              value={url}
              onFocus={(event) => event.target.select()}
            />
          </label>
          <button type="submit">Done</button>
        </form>
      </dialog>
    </div>
  );
}
