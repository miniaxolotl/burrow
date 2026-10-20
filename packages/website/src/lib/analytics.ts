"use client";

import posthog from "posthog-js";
import {
  analyticsEvents,
  maskReplayAttribute,
  pagePath,
  safePageUrl,
  sanitizeEvent,
  type AnalyticsEvent,
  type Placement,
} from "./analytics-policy";
import {
  activateTelemetryGeneration,
  browserOptOut,
  notifyPreference,
  storePreference,
  subscribePreference,
  telemetryOptedOut,
  telemetryAllowed,
} from "./telemetry-choice";

let initialized = false;
let suspended = false;
let lastPage: string | undefined;
function registerContext() {
  posthog.register({
    product: "burrow",
    surface: "website",
    environment: "production",
    version: process.env.NEXT_PUBLIC_APP_VERSION,
    release: process.env.NEXT_PUBLIC_TELEMETRY_RELEASE,
  });
  posthog.setPersonPropertiesForFlags(
    { environment: "production", surface: "website" },
    false,
  );
}

export function initializeAnalytics() {
  if (
    initialized ||
    telemetryOptedOut() ||
    process.env.NEXT_PUBLIC_DEPLOYMENT_ENVIRONMENT !== "production" ||
    process.env.NEXT_PUBLIC_POSTHOG_ENABLED !== "true" ||
    !process.env.NEXT_PUBLIC_POSTHOG_KEY
  )
    return;
  try {
    activateTelemetryGeneration();
    posthog.init(process.env.NEXT_PUBLIC_POSTHOG_KEY, {
      api_host: "https://us.i.posthog.com",
      ui_host: "https://us.posthog.com",
      defaults: "2026-01-30",
      persistence: "localStorage",
      opt_out_persistence_by_default: true,
      cross_subdomain_cookie: false,
      person_profiles: "never",
      ip: false,
      respect_dnt: true,
      autocapture: false,
      capture_pageview: false,
      capture_pageleave: true,
      capture_exceptions: false,
      capture_heatmaps: false,
      capture_performance: { web_vitals: true, network_timing: false },
      enable_recording_console_log: false,
      disable_session_recording: true,
      disable_surveys: false,
      advanced_disable_feature_flags_on_first_load: true,
      session_recording: {
        maskAllInputs: true,
        maskTextSelector: "*",
        blockSelector:
          "img, video, canvas, iframe, [data-private], .ph-no-capture",
        maskAttributeFn: maskReplayAttribute,
        captureJsonLd: false,
        recordCrossOriginIframes: false,
        collectFonts: false,
        recordHeaders: false,
        recordBody: false,
        maskCapturedNetworkRequestFn: (request) => ({
          ...request,
          name: safePageUrl(request.name),
        }),
      },
      before_send: (event) =>
        !telemetryAllowed()
          ? null
          : sanitizeEvent(event, location.href, document.referrer),
      loaded: (client) => {
        registerContext();
        client.onFeatureFlags(() => {
          if (telemetryAllowed())
            client.set_config({
              disable_session_recording:
                client.getFeatureFlag("session-replay", {
                  send_event: false,
                }) !== true,
            });
        });
        client.reloadFeatureFlags();
      },
    });
    initialized = true;
    subscribePreference(() => {
      if (!telemetryAllowed() && !suspended) {
        suspended = true;
        posthog.opt_out_capturing();
        posthog.stopSessionRecording();
      }
    });
  } catch {
    /* Telemetry never prevents rendering. */
  }
}

export function setTelemetryOptOut(disabled: boolean) {
  try {
    storePreference(disabled);
    if (disabled && initialized) {
      suspended = true;
      posthog.opt_out_capturing();
      posthog.stopSessionRecording();
    } else if (!disabled && !browserOptOut()) {
      suspended = false;
      activateTelemetryGeneration();
      if (initialized) posthog.reset(true);
      else initializeAnalytics();
      if (initialized) {
        posthog.opt_in_capturing({ captureEventName: false });
        registerContext();
        posthog.reloadFeatureFlags();
      }
      lastPage = undefined;
      capturePageview(location.pathname);
    }
  } catch {
    /* Storage failure keeps collection disabled. */
  } finally {
    notifyPreference();
  }
}

export function capture(
  event: AnalyticsEvent,
  properties: {
    target_product?: string;
    placement?: Placement;
    operation?: "copy_install_command" | "copy_example_command";
    outcome?: "success" | "unavailable" | "error";
    duration_ms?: number;
  } = {},
) {
  if (!initialized || !telemetryAllowed()) return;
  try {
    posthog.capture(event, properties);
  } catch {
    /* Best effort. */
  }
}
export function capturePageview(path: string) {
  initializeAnalytics();
  if (!initialized || !telemetryAllowed()) return;
  const route = pagePath(path);
  if (route === lastPage) return;
  lastPage = route;
  try {
    posthog.capture("$pageview", { $current_url: safePageUrl(path) });
  } catch {
    /* Best effort. */
  }
}
export function subscribeCtaVariant(callback: () => void) {
  let unsubscribeFlags: (() => void) | undefined;
  const update = () => {
    unsubscribeFlags?.();
    unsubscribeFlags =
      initialized && telemetryAllowed()
        ? posthog.onFeatureFlags(callback)
        : undefined;
    callback();
  };
  update();
  const unsubscribePreference = subscribePreference(update);
  return () => {
    unsubscribeFlags?.();
    unsubscribePreference();
  };
}
export function ctaVariant(): "control" | "benefit" {
  return initialized &&
    telemetryAllowed() &&
    posthog.getFeatureFlag("website-cta-copy", { send_event: false }) ===
      "benefit"
    ? "benefit"
    : "control";
}
export function captureCtaExposure() {
  if (initialized && telemetryAllowed())
    posthog.getFeatureFlag("website-cta-copy");
}
export function captureLinkClick(event: MouseEvent) {
  if (!(event.target instanceof Element)) return;
  const link = event.target.closest<HTMLElement>("[data-analytics-event]");
  const name = link?.dataset.analyticsEvent;
  if (!name || !(analyticsEvents as readonly string[]).includes(name)) return;
  capture(name as AnalyticsEvent, {
    target_product: link.dataset.analyticsProduct,
    placement: link.dataset.analyticsPlacement as Placement,
  });
}
