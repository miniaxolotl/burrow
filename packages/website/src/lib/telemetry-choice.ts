const preferenceKey = "burrow.telemetry.optout";
const preferenceEvent = "burrow:telemetry-preference";

export function browserOptOut(): boolean {
  return (
    typeof navigator !== "undefined" &&
    (navigator.doNotTrack === "1" ||
      (navigator as Navigator & { globalPrivacyControl?: boolean })
        .globalPrivacyControl === true)
  );
}
export function telemetryOptedOut(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return browserOptOut() || localStorage.getItem(preferenceKey) === "true";
  } catch {
    return true;
  }
}
export function subscribePreference(callback: () => void) {
  window.addEventListener(preferenceEvent, callback);
  window.addEventListener("storage", callback);
  return () => {
    window.removeEventListener(preferenceEvent, callback);
    window.removeEventListener("storage", callback);
  };
}
export function storePreference(disabled: boolean) {
  localStorage.setItem(preferenceKey, String(disabled));
}
export function notifyPreference() {
  window.dispatchEvent(new Event(preferenceEvent));
}
