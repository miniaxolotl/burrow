"use client";

import * as Sentry from "@sentry/nextjs";
import {
  dataCollection,
  sanitizeError,
  sanitizeLog,
  sanitizeMetric,
  sanitizeSpan,
} from "./diagnostics-policy";
import { subscribePreference, telemetryOptedOut } from "./telemetry-choice";

let initialized = false;
export function initializeDiagnostics() {
  if (
    initialized ||
    telemetryOptedOut() ||
    process.env.NEXT_PUBLIC_DEPLOYMENT_ENVIRONMENT !== "production" ||
    process.env.NEXT_PUBLIC_SENTRY_ENABLED !== "true" ||
    !process.env.NEXT_PUBLIC_SENTRY_DSN
  )
    return;
  try {
    Sentry.init({
      dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
      environment: "production",
      release: process.env.NEXT_PUBLIC_TELEMETRY_RELEASE,
      dataCollection,
      sendClientReports: false,
      integrations: [Sentry.browserTracingIntegration()],
      tracesSampleRate: 0.1,
      tracePropagationTargets: [/^https:\/\/useburrow\.app\//],
      beforeBreadcrumb: () => null,
      beforeSend: (event) =>
        telemetryOptedOut() ? null : sanitizeError(event),
      beforeSendSpan: sanitizeSpan,
      beforeSendLog: (log) => (telemetryOptedOut() ? null : sanitizeLog(log)),
      beforeSendMetric: (metric) =>
        telemetryOptedOut() ? null : sanitizeMetric(metric),
      // Covers every envelope type, including streamed spans and queued batches.
      transport: (options) => {
        const transport = Sentry.makeFetchTransport(options);
        return {
          ...transport,
          send: (envelope) =>
            telemetryOptedOut()
              ? Promise.resolve({ statusCode: 200 })
              : transport.send(envelope),
        };
      },
    } satisfies Sentry.BrowserOptions);
    initialized = true;
  } catch {
    /* Diagnostics are independent of product behavior. */
  }
}
export function observeDiagnosticsPreference() {
  initializeDiagnostics();
  subscribePreference(initializeDiagnostics);
}
export function recordClipboardOutcome(
  operation: "copy_install_command" | "copy_example_command",
  outcome: "success" | "unavailable",
  duration: number,
) {
  if (!initialized || telemetryOptedOut()) return;
  try {
    const attributes = { operation, outcome };
    Sentry.logger.info("Clipboard operation completed", attributes);
    Sentry.metrics.count("website.operation.count", 1, { attributes });
    Sentry.metrics.distribution("website.operation.duration", duration, {
      unit: "millisecond",
      attributes,
    });
  } catch {
    /* Best effort. */
  }
}
