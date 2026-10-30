/**
 * API client for the Go (Cloud Run) backend.
 *
 * Contract consumed (must match the api/ layer):
 *
 *   POST {API_BASE}/generate
 *     body: { "input": string }            // server re-validates non-empty + <= MAX_INPUT_BYTES
 *     200:  text/event-stream (SSE), named events:
 *             event: token   data: {"text":"<chunk>"}     // 0..n, append in order
 *             event: done    data: {}                       // stream finished cleanly
 *             event: error   data: {"error":"<message>"}    // mid-stream failure
 *     429:  JSON { "error": "rate_limited" }                // before stream starts
 *     400:  JSON { "error": "<validation message>" }
 *
 *   POST {API_BASE}/signal
 *     body: { "kind": "email", "email": string }                // optional "inputExcerpt"
 *        or { "kind": "thumbs", "thumbs": "up" | "down" }        // optional "inputExcerpt"
 *     201:  JSON { "status": "recorded" }                        // stored
 *     400:  JSON { "error": "<message>" }                        // validation
 *
 * API_BASE comes from NEXT_PUBLIC_API_BASE (set per-slug via Terraform/Vercel env).
 * Falls back to same-origin "" so a local proxy/rewrite also works.
 */

export const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

export type GenerateEvent =
  | { type: "token"; text: string }
  | { type: "done" }
  | { type: "error"; error: string };

export class RateLimitError extends Error {
  constructor() {
    super("rate_limited");
    this.name = "RateLimitError";
  }
}

/**
 * POST /generate and yield SSE events as they arrive. Streams via the fetch
 * Response body reader, parsing the text/event-stream framing incrementally.
 */
export async function* streamGenerate(
  input: string,
  signal?: AbortSignal,
): AsyncGenerator<GenerateEvent> {
  const res = await fetch(`${API_BASE}/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "text/event-stream" },
    body: JSON.stringify({ input }),
    signal,
  });

  if (res.status === 429) throw new RateLimitError();
  if (!res.ok || !res.body) {
    let message = `request failed (${res.status})`;
    try {
      const data = (await res.json()) as { error?: string };
      if (data?.error) message = data.error;
    } catch {
      // non-JSON body; keep the default message
    }
    throw new Error(message);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });

      // SSE frames are separated by a blank line.
      let sep: number;
      while ((sep = buffer.indexOf("\n\n")) !== -1) {
        const rawFrame = buffer.slice(0, sep);
        buffer = buffer.slice(sep + 2);
        const evt = parseFrame(rawFrame);
        if (evt) yield evt;
      }
    }
    // Flush any trailing frame without a terminating blank line.
    const tail = parseFrame(buffer);
    if (tail) yield tail;
  } finally {
    reader.releaseLock();
  }
}

function parseFrame(frame: string): GenerateEvent | null {
  let event = "message";
  const dataLines: string[] = [];

  for (const line of frame.split("\n")) {
    if (line === "" || line.startsWith(":")) continue; // blank / comment
    const colon = line.indexOf(":");
    const field = colon === -1 ? line : line.slice(0, colon);
    // Per spec, a single leading space after the colon is stripped.
    let val = colon === -1 ? "" : line.slice(colon + 1);
    if (val.startsWith(" ")) val = val.slice(1);
    if (field === "event") event = val;
    else if (field === "data") dataLines.push(val);
  }

  const data = dataLines.join("\n");
  if (event === "token") {
    try {
      const parsed = JSON.parse(data) as { text?: string };
      return { type: "token", text: parsed.text ?? "" };
    } catch {
      // Tolerate a raw-text token payload.
      return { type: "token", text: data };
    }
  }
  if (event === "done") return { type: "done" };
  if (event === "error") {
    try {
      const parsed = JSON.parse(data) as { error?: string };
      return { type: "error", error: parsed.error ?? "generation failed" };
    } catch {
      return { type: "error", error: data || "generation failed" };
    }
  }
  return null;
}

export type SignalBody =
  | { kind: "email"; email: string; inputExcerpt?: string }
  | { kind: "thumbs"; thumbs: "up" | "down"; inputExcerpt?: string };

/**
 * POST /signal — the validation payoff. Returns true when stored.
 * The server replies 201 { "status": "recorded" } on success.
 */
export async function postSignal(body: SignalBody): Promise<boolean> {
  const res = await fetch(`${API_BASE}/signal`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) return false;
  try {
    const data = (await res.json()) as { status?: string };
    return data?.status === "recorded";
  } catch {
    // 2xx with an unparseable/empty body still counts as stored.
    return res.ok;
  }
}
