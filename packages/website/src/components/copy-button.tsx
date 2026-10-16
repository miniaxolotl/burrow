"use client";

import { Check, Copy } from "lucide-react";
import { useEffect, useState } from "react";

export function CopyButton({
  text,
  label = "Copy command",
}: {
  text: string;
  label?: string;
}) {
  const [status, setStatus] = useState<"idle" | "copied" | "error">("idle");

  useEffect(() => {
    if (status === "idle") return;
    const timeout = window.setTimeout(() => setStatus("idle"), 3500);
    return () => window.clearTimeout(timeout);
  }, [status]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setStatus("copied");
    } catch {
      setStatus("error");
    }
  }

  return (
    <span className="relative inline-flex shrink-0">
      <button
        type="button"
        onClick={copy}
        className="copy-button"
        aria-label={status === "copied" ? "Copied to clipboard" : label}
        title={status === "copied" ? "Copied!" : label}
      >
        {status === "copied" ? <Check size={16} /> : <Copy size={16} />}
      </button>
      <span
        role="status"
        className={status === "error" ? "copy-feedback" : "sr-only"}
      >
        {status === "copied" && "Copied to clipboard."}
        {status === "error" &&
          "Clipboard unavailable. Select and copy the command instead."}
      </span>
    </span>
  );
}
