"use client";

import { useEffect } from "react";
import { consentAllowed, subscribeConsent } from "@/lib/cookie-consent";

const containerId = "GTM-PGLVNMZ8";
const hosts = new Set(["useburrow.app", "www.useburrow.app"]);
const paths = new Set(["/", "/privacy", "/terms"]);
let started = false;

function publicLocation(url: URL) {
  return (
    url.protocol === "https:" &&
    hosts.has(url.hostname) &&
    paths.has(url.pathname) &&
    [...url.searchParams.keys()].every((key) =>
      /^utm_(source|medium|campaign|term|content)$/.test(key),
    )
  );
}

export function WebsiteTagManager() {
  useEffect(() => {
    const initialLocation = new URL(location.href);
    if (initialLocation.searchParams.get("tracking") === "off")
      initialLocation.searchParams.delete("tracking");
    // Only the public website may load tags, never tunnel subdomains or previews.
    if (
      process.env.NEXT_PUBLIC_DEPLOYMENT_ENVIRONMENT !== "production" ||
      !publicLocation(initialLocation)
    )
      return;
    const update = () => {
      if (!consentAllowed("advertising")) {
        if (started) {
          // Unload vendor code even if saving the local preference failed.
          const url = new URL(location.href);
          url.searchParams.set("tracking", "off");
          location.replace(url.href);
        }
        return;
      }
      if (started || !publicLocation(new URL(location.href))) return;
      started = true;
      const browser = window as typeof window & { dataLayer?: unknown[] };
      browser.dataLayer ??= [];
      browser.dataLayer.push({ "gtm.start": Date.now(), event: "gtm.js" });
      const script = document.createElement("script");
      script.id = "website-tag-manager";
      script.async = true;
      script.src = `https://www.googletagmanager.com/gtm.js?id=${containerId}`;
      document.head.appendChild(script);
    };
    const navigate = (event: MouseEvent) => {
      if (
        !started ||
        event.defaultPrevented ||
        event.button !== 0 ||
        event.metaKey ||
        event.ctrlKey ||
        event.shiftKey ||
        event.altKey
      )
        return;
      const link =
        event.target instanceof Element
          ? event.target.closest<HTMLAnchorElement>("a[href]")
          : null;
      if (
        !link ||
        link.hasAttribute("download") ||
        (link.target && link.target !== "_self")
      )
        return;
      const url = new URL(link.href);
      if (
        url.origin === location.origin &&
        (url.pathname !== location.pathname || url.search !== location.search)
      ) {
        // Reset vendor state and fire DOM Ready once for each new public page.
        event.preventDefault();
        location.assign(url.href);
      }
    };
    update();
    const unsubscribe = subscribeConsent(update);
    document.addEventListener("click", navigate, true);
    return () => {
      unsubscribe();
      document.removeEventListener("click", navigate, true);
    };
  }, []);
  return null;
}
