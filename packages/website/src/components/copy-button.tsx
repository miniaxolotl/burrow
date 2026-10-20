"use client";

import { Check, Copy } from "lucide-react";
import { useEffect, useState } from "react";
import { capture } from "@/lib/analytics";
import { recordClipboardOutcome } from "@/lib/diagnostics";

export function CopyButton({
  text,
  label = "Copy command",
  installPlacement,
}: {
  text: string;
  label?: string;
  installPlacement?: "hero" | "install";
}) {
  const [status, setStatus] = useState<"idle" | "copied" | "error">("idle");

  useEffect(() => {
    if (status === "idle") return;
    const timeout = window.setTimeout(() => setStatus("idle"), 3500);
    return () => window.clearTimeout(timeout);
  }, [status]);

  async function copy() {
    const started = performance.now();
    const operation = installPlacement
      ? "copy_install_command"
      : "copy_example_command";
    try {
      await navigator.clipboard.writeText(text);
      setStatus("copied");
      if (installPlacement)
        capture("install_command_copied", { placement: installPlacement });
      capture("operation_completed", {
        operation,
        outcome: "success",
        duration_ms: performance.now() - started,
      });
      recordClipboardOutcome(operation, "success", performance.now() - started);
    } catch {
      setStatus("error");
      capture("operation_completed", {
        operation,
        outcome: "unavailable",
        duration_ms: performance.now() - started,
      });
      recordClipboardOutcome(
        operation,
        "unavailable",
        performance.now() - started,
      );
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
