"use client";

/**
 * SignalCapture — the validation payoff, shown AFTER the user sees value.
 *
 * Renders signalQuestion + the configured signal kind:
 *   - "email":  input + consentCopy -> POST /signal { kind:"email", email }
 *   - "thumbs": up/down            -> POST /signal { kind:"thumbs", thumbs }
 * On a stored signal, fires onCaptured so the page emits poc_signal.
 * Anonymous, no auth. >=44px controls. Email validated client-side; the server
 * re-validates (authoritative).
 */

import { useState, useCallback } from "react";
import { ThumbsUp, ThumbsDown, Send, Check, Loader2 } from "lucide-react";
import { postSignal } from "../lib/apiClient";
import type { ValidationSignal } from "../../idea.config.types";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function SignalCapture({
  question,
  signal,
  onCaptured,
}: {
  question: string;
  signal: ValidationSignal;
  /** Called once after the signal is stored, with the captured detail. */
  onCaptured?: (detail: { kind: "email" } | { kind: "thumbs"; thumbs: "up" | "down" }) => void;
}) {
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<"idle" | "sending" | "done" | "error">("idle");
  const [error, setError] = useState<string | null>(null);

  const submitEmail = useCallback(async () => {
    const value = email.trim();
    if (!EMAIL_RE.test(value)) {
      setError("Enter a valid email address.");
      return;
    }
    setError(null);
    setStatus("sending");
    const ok = await postSignal({ kind: "email", email: value });
    if (ok) {
      setStatus("done");
      onCaptured?.({ kind: "email" });
    } else {
      setStatus("error");
      setError("Couldn't save that — please try again.");
    }
  }, [email, onCaptured]);

  const submitThumbs = useCallback(
    async (thumbs: "up" | "down") => {
      setError(null);
      setStatus("sending");
      const ok = await postSignal({ kind: "thumbs", thumbs });
      if (ok) {
        setStatus("done");
        onCaptured?.({ kind: "thumbs", thumbs });
      } else {
        setStatus("error");
        setError("Couldn't save that — please try again.");
      }
    },
    [onCaptured],
  );

  if (status === "done") {
    return (
      <section className="signal signal-done" aria-live="polite">
        <Check size={20} aria-hidden />
        <p>Thanks — that's the signal we needed.</p>
      </section>
    );
  }

  return (
    <section className="signal">
      <p className="signal-question">{question}</p>

      {signal.kind === "email" ? (
        <form
          className="signal-email"
          onSubmit={(e) => {
            e.preventDefault();
            void submitEmail();
          }}
        >
          <label className="signal-consent" htmlFor="signal-email">
            {signal.consentCopy}
          </label>
          <div className="signal-row">
            <input
              id="signal-email"
              type="email"
              inputMode="email"
              autoComplete="email"
              className="signal-input"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
            <button type="submit" className="btn btn-primary" disabled={status === "sending"}>
              {status === "sending" ? (
                <Loader2 size={18} className="spin" aria-hidden />
              ) : (
                <Send size={18} aria-hidden />
              )}
              <span>Send it to me</span>
            </button>
          </div>
        </form>
      ) : (
        <div className="signal-thumbs" role="group" aria-label={question}>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => void submitThumbs("up")}
            disabled={status === "sending"}
            aria-label="Yes, I'd use this"
          >
            <ThumbsUp size={20} aria-hidden />
            <span>Yes</span>
          </button>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => void submitThumbs("down")}
            disabled={status === "sending"}
            aria-label="No, I wouldn't"
          >
            <ThumbsDown size={20} aria-hidden />
            <span>No</span>
          </button>
        </div>
      )}

      {error && (
        <p className="signal-error" role="alert">
          {error}
        </p>
      )}
    </section>
  );
}
