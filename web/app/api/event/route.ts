/**
 * Thin first-party analytics sink. Receives the funnel beacons from
 * components/analytics.ts and forwards them to whatever backend is wired per
 * deploy (ANALYTICS_FORWARD_URL), or just acks (204) when none is set — the
 * platform-native sink on the client may already capture them.
 *
 * Free / no-lock-in by design: no SDK, no hard dependency. Never blocks the
 * client; failures are swallowed.
 */

export const runtime = "edge";

export async function POST(req: Request): Promise<Response> {
  try {
    const body = await req.text();
    const forwardUrl = process.env.ANALYTICS_FORWARD_URL;
    if (forwardUrl) {
      // Fire-and-forget forward; don't make the client wait on a slow backend.
      void fetch(forwardUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body,
      }).catch(() => {});
    }
  } catch {
    // ignore malformed beacons
  }
  return new Response(null, { status: 204 });
}
