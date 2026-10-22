"use client";

import { useEffect, useSyncExternalStore } from "react";
import { usePathname } from "next/navigation";
import { CookieSettings } from "./cookie-consent";
import { consentAllowed, showCookieSettings } from "@/lib/cookie-consent";
import {
  captureCtaExposure,
  captureLinkClick,
  capturePageview,
  ctaVariant,
  setTelemetryOptOut,
  subscribeCtaVariant,
} from "@/lib/analytics";
import {
  browserOptOut,
  subscribePreference,
  telemetryOptedOut,
} from "@/lib/telemetry-choice";

export function WebsiteTelemetry() {
  const pathname = usePathname();
  useEffect(() => {
    capturePageview(pathname);
  }, [pathname]);
  useEffect(() => {
    document.addEventListener("click", captureLinkClick);
    return () => document.removeEventListener("click", captureLinkClick);
  }, []);
  return null;
}
export function TelemetryChoice() {
  const optedOut = useSyncExternalStore(
    subscribePreference,
    telemetryOptedOut,
    () => false,
  );
  return (
    <>
      <CookieSettings />
      <button
        type="button"
        className="text-link"
        disabled={optedOut && browserOptOut()}
        onClick={() =>
          optedOut && !consentAllowed("analytics")
            ? showCookieSettings()
            : setTelemetryOptOut(!optedOut)
        }
      >
        {optedOut
          ? browserOptOut()
            ? "Telemetry disabled by browser"
            : "Enable website telemetry"
          : "Disable website telemetry"}
      </button>
    </>
  );
}
export function HeroCtaCopy() {
  const variant = useSyncExternalStore(
    subscribeCtaVariant,
    ctaVariant,
    () => "control",
  );
  useEffect(() => {
    captureCtaExposure();
  }, [variant]);
  return variant === "benefit" ? "Share your local app" : "Install Burrow";
}
