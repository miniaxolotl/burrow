import * as Sentry from "@sentry/nextjs";
import { initializeAnalytics } from "./lib/analytics";
import { observeDiagnosticsPreference } from "./lib/diagnostics";

initializeAnalytics();
observeDiagnosticsPreference();
export const onRouterTransitionStart = Sentry.captureRouterTransitionStart;
