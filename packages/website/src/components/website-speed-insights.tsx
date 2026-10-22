"use client";

import { SpeedInsights } from "@vercel/speed-insights/next";
import { Analytics } from "@vercel/analytics/next";
import { useSyncExternalStore } from "react";
import { telemetryOptedOut, subscribePreference } from "@/lib/telemetry-choice";

const publicPaths = new Set(["/", "/privacy", "/terms"]);

export function WebsiteSpeedInsights() {
  const disabled = useSyncExternalStore(
    subscribePreference,
    telemetryOptedOut,
    () => true,
  );
  if (disabled) return null;
  return (
    <SpeedInsights
      beforeSend={(data) => {
        if (telemetryOptedOut()) return null;
        const url = new URL(data.url);
        if (!publicPaths.has(url.pathname)) return null;
        return {
          ...data,
          url: `${url.origin}${url.pathname}`,
          route: url.pathname,
        };
      }}
    />
  );
}

export function WebsiteVercelAnalytics() {
  const disabled = useSyncExternalStore(
    subscribePreference,
    telemetryOptedOut,
    () => true,
  );
  return disabled ? null : (
    <Analytics beforeSend={(event) => (telemetryOptedOut() ? null : event)} />
  );
}
