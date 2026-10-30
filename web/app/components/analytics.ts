/**
 * The validation funnel — the experiment result.
 *
 * The 4 funnel events, emitted by construction (see README funnel table):
 *   poc_visit | poc_generate | poc_output_action | poc_signal
 * Headline metric = poc_signal / poc_visit. Namespaced by the config slug so a
 * deployment's events are isolated.
 *
 * Sink: free + no-lock-in. Posts a small JSON beacon to /api/event (a thin
 * same-origin route that forwards to whatever analytics backend is wired per
 * deploy) and also mirrors to any platform-native sink on window. Never throws
 * — analytics must not break the core flow.
 */

import { ideaConfig } from "../../idea.config";

export type FunnelEvent = "poc_visit" | "poc_generate" | "poc_output_action" | "poc_signal";

declare global {
  interface Window {
    // Optional platform-native sink (e.g. Vercel Analytics / Plausible custom event).
    // Left untyped-loose on purpose: present only if a provider script is loaded.
    va?: (event: "event", props: Record<string, unknown>) => void;
  }
}

export function track(event: FunnelEvent, props?: Record<string, unknown>): void {
  const payload = {
    event,
    slug: ideaConfig.slug,
    ts: Date.now(),
    ...props,
  };

  try {
    // Platform-native sink, if a provider is present.
    if (typeof window !== "undefined" && typeof window.va === "function") {
      window.va("event", { name: event, ...payload });
    }

    // First-party beacon — survives page unload (sendBeacon), best-effort.
    if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
      const blob = new Blob([JSON.stringify(payload)], { type: "application/json" });
      navigator.sendBeacon("/api/event", blob);
      return;
    }

    if (typeof fetch !== "undefined") {
      void fetch("/api/event", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
        keepalive: true,
      }).catch(() => {});
    }
  } catch {
    // Analytics must never break the core flow.
  }
}
