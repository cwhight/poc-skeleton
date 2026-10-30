"use client";

/**
 * The core POC flow: paste -> generate (streamed) -> output + signal.
 *
 * Funnel wiring (by construction):
 *   - poc_generate        on submit (the paid call)
 *   - poc_output_action   on copy/download from OutputView
 *   - poc_signal          on a captured signal from SignalCapture
 * (poc_visit is emitted on the landing page.)
 *
 * Client-side input cap mirrors config.maxInputBytes as a BYTE length (not
 * string length) — the Go server is authoritative and re-checks, this is a UX
 * hint that prevents an obviously-too-big request. Streamed tokens render
 * incrementally through OutputView (escape-then-markdown, the single safe path).
 * Vendor-neutral copy; >=44px controls; mobile-first.
 */

import { useMemo, useRef, useState, useCallback } from "react";
import Link from "next/link";
import { ArrowLeft, Sparkles, Loader2 } from "lucide-react";
import { ideaConfig } from "../../idea.config";
import { track } from "../components/analytics";
import { OutputView } from "../components/OutputView";
import { SignalCapture } from "../components/SignalCapture";
import { streamGenerate, RateLimitError } from "../lib/apiClient";

const encoder = new TextEncoder();

export default function GeneratePage() {
  const [input, setInput] = useState("");
  const [output, setOutput] = useState("");
  const [phase, setPhase] = useState<"idle" | "streaming" | "done">("idle");
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const byteLen = useMemo(() => encoder.encode(input).length, [input]);
  const overLimit = byteLen > ideaConfig.maxInputBytes;
  const canSubmit = input.trim().length > 0 && !overLimit && phase !== "streaming";

  const handleGenerate = useCallback(async () => {
    if (!canSubmit) return;
    track("poc_generate", { bytes: byteLen });

    setError(null);
    setOutput("");
    setPhase("streaming");

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      for await (const evt of streamGenerate(input, controller.signal)) {
        if (evt.type === "token") {
          setOutput((prev) => prev + evt.text);
        } else if (evt.type === "error") {
          throw new Error(evt.error);
        }
        // "done" ends the generator loop naturally.
      }
      setPhase("done");
    } catch (err) {
      if (controller.signal.aborted) return;
      if (err instanceof RateLimitError) {
        setError("You've hit the rate limit for now — give it a minute and try again.");
      } else {
        setError(err instanceof Error ? err.message : "Something went wrong. Try again.");
      }
      // Keep any partial output visible; mark the run finished.
      setPhase(output ? "done" : "idle");
    } finally {
      abortRef.current = null;
    }
  }, [canSubmit, input, byteLen, output]);

  const handleOutputAction = useCallback((action: "copy" | "download") => {
    track("poc_output_action", { action });
  }, []);

  const handleSignal = useCallback(
    (detail: { kind: "email" } | { kind: "thumbs"; thumbs: "up" | "down" }) => {
      track("poc_signal", detail);
    },
    [],
  );

  const showOutput = output.length > 0;
  const showSignal = phase === "done" && showOutput;

  return (
    <main>
      <Link href="/" className="gen-back">
        <ArrowLeft size={18} aria-hidden />
        <span>Back</span>
      </Link>

      <h2 className="gen-title">{ideaConfig.name}</h2>

      <label className="field-label" htmlFor="poc-input">
        {ideaConfig.inputLabel}
      </label>
      <textarea
        id="poc-input"
        className="input"
        placeholder={ideaConfig.placeholder}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        disabled={phase === "streaming"}
      />
      <div className={`input-meta${overLimit ? " over" : ""}`}>
        <span>
          {byteLen.toLocaleString()} / {ideaConfig.maxInputBytes.toLocaleString()} bytes
        </span>
        {overLimit && <span>Too long — trim the input.</span>}
      </div>

      <button
        type="button"
        className="btn btn-primary gen-submit"
        onClick={() => void handleGenerate()}
        disabled={!canSubmit}
      >
        {phase === "streaming" ? (
          <Loader2 size={18} className="spin" aria-hidden />
        ) : (
          <Sparkles size={18} aria-hidden />
        )}
        <span>{phase === "streaming" ? "Generating…" : "Generate"}</span>
      </button>

      {error && (
        <div className="error-banner" role="alert">
          {error}
        </div>
      )}

      {showOutput && (
        <OutputView
          text={output}
          format={ideaConfig.outputFormat}
          ctaText={ideaConfig.ctaText}
          onAction={handleOutputAction}
        />
      )}

      {showSignal && (
        <SignalCapture
          question={ideaConfig.signalQuestion}
          signal={ideaConfig.signal}
          onCaptured={handleSignal}
        />
      )}
    </main>
  );
}
