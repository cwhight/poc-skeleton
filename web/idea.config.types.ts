/**
 * idea.config schema — mirror of ../idea.config.example.ts (kept inside web/ so
 * the Next.js app is self-contained and builds without importing above its root).
 * The canonical schema + docs live in templates/poc-skeleton/idea.config.example.ts.
 */

export type OutputFormat = "markdown" | "plaintext";

export type ValidationSignal =
  | { kind: "email"; consentCopy: string } // capture an email ("send me this / notify me")
  | { kind: "thumbs" }; // thumbs up/down only

export interface IdeaConfig {
  /** Human-readable product name, e.g. "ScopeLock". */
  name: string;
  /** Stable deploy identifier; namespaces service/DB/analytics. Lowercase, no spaces. */
  slug: string;
  /** One-line positioning, used in metadata and the landing hero. */
  tagline: string;

  /** Landing hero headline — the promise. */
  landingHeadline: string;
  /** Landing sub-copy — who it's for + the value, honest and verifiable. */
  subcopy: string;

  /** Label above the paste box. */
  inputLabel: string;
  /** Placeholder showing the kind of input expected. */
  placeholder: string;
  /** Hard server-side cap on input size (bytes). Mirrors MAX_INPUT_BYTES. */
  maxInputBytes: number;

  /** THE generate prompt. Sent as the system prompt; user input is the message. */
  systemPrompt: string;

  /** How the model output is rendered + offered for take-away. */
  outputFormat: OutputFormat;

  /** Primary action button after generation, e.g. "Copy contract". */
  ctaText: string;

  /** The validation question / signal shown after the user sees value. */
  signalQuestion: string;
  /** How the signal is captured. */
  signal: ValidationSignal;

  /** The riskiest "would they trust/use this" assumption. Not rendered. */
  hypothesis: string;
  /** Headline success threshold, e.g. "poc_signal / poc_visit >= 8%". */
  successMetric: string;
  /** When to kill the idea. */
  killCondition: string;
}
