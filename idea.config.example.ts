/**
 * idea.config — THE single artifact a new POC fills in.
 *
 * This file *is* the product definition. Everything else in the skeleton is
 * generic and reads from here. To create a new POC:
 *   1. Copy this file to `idea.config.ts` and fill every field.
 *   2. Iterate `systemPrompt` against real inputs until the output is something
 *      a user would actually copy/download.
 *   3. `slug` parameterizes deploy (Vercel project, Cloud Run service, Neon DB,
 *      analytics namespace) — keep it stable, lowercase, hyphen-free.
 *
 * See worked examples: configs/scopelock.config.ts, configs/reviewpilot.config.ts
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

  /**
   * The experiment, decided BEFORE launch (kept honest — see PATTERNS-BAKED-IN).
   * Not rendered; read by the team and the funnel dashboard.
   */
  hypothesis: string;
  /** Headline success threshold, e.g. "poc_signal / poc_visit >= 8%". */
  successMetric: string;
  /** When to kill the idea, e.g. "<3% signal after 300 visits / 14 days". */
  killCondition: string;
}

// ---------------------------------------------------------------------------
// EXAMPLE (fill this in for your idea). Two complete worked examples live in
// configs/scopelock.config.ts and configs/reviewpilot.config.ts.
// ---------------------------------------------------------------------------
export const ideaConfig: IdeaConfig = {
  name: "Acme POC",
  slug: "acme-poc",
  tagline: "<one-line positioning>",

  landingHeadline: "<the promise, in the user's words>",
  subcopy: "<who it's for + the value — honest, no invented metrics>",

  inputLabel: "Paste your <thing>",
  placeholder: "<example of the kind of input expected>",
  maxInputBytes: 20_000,

  systemPrompt: [
    "You are <role>. Turn the user's <input> into <output>.",
    "Rules: <constraints, tone, what to include/exclude>.",
    "Output <format>. Do not add commentary before or after.",
  ].join("\n"),

  outputFormat: "markdown",
  ctaText: "Copy result",

  signalQuestion: "Would you use this for real work?",
  signal: {
    kind: "email",
    consentCopy: "Email it to me and tell me when it launches.",
  },

  hypothesis: "<the riskiest 'would they trust/use this' assumption>",
  successMetric: "poc_signal / poc_visit >= <X>%",
  killCondition: "<signal below Y% after N visits / D days>",
};
