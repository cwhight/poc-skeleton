/**
 * Worked example — ScopeLock.
 * Freelancer pastes a rough project brief → gets a clean, scoped contract.
 * Riskiest assumption: do freelancers trust an AI-drafted contract enough to use it?
 */

import type { IdeaConfig } from "../idea.config.example";

export const ideaConfig: IdeaConfig = {
  name: "ScopeLock",
  slug: "scopelock",
  tagline: "Turn a messy project brief into a freelance contract you can send today.",

  landingHeadline: "Paste your brief. Get a contract that protects your scope.",
  subcopy:
    "For freelancers who win the work, then dread writing the contract. ScopeLock turns a few lines about the job into a clear scope, deliverables, payment terms, and a revision cap — so 'just one more change' doesn't eat your margin.",

  inputLabel: "Describe the project (what, for whom, rough price & timeline)",
  placeholder:
    "Logo + brand guidelines for a coffee roaster. ~3 weeks, £1,800, 2 rounds of revisions. They want social templates too.",
  maxInputBytes: 12_000,

  systemPrompt: [
    "You are a contracts assistant for independent freelancers.",
    "Turn the user's informal project brief into a clear, fair fixed-scope freelance contract.",
    "Include these sections: Parties, Scope of Work, Deliverables, Out of Scope, Timeline & Milestones, Fees & Payment Schedule, Revisions (cap explicitly), Intellectual Property (transfers on final payment), Termination & Kill Fee, and a signature block.",
    "Infer sensible terms from the brief; where a detail is missing, insert a clearly bracketed placeholder like [CLIENT NAME] rather than inventing facts.",
    "Plain, firm language a non-lawyer can understand. Protect the freelancer's scope and cashflow without being adversarial.",
    "Add a one-line disclaimer that this is a starting template, not legal advice.",
    "Output GitHub-flavoured Markdown only. No commentary before or after the contract.",
  ].join("\n"),

  outputFormat: "markdown",
  ctaText: "Copy contract",

  signalQuestion: "Would you send a contract like this to a real client?",
  signal: {
    kind: "email",
    consentCopy: "Email me this contract (and tell me when ScopeLock launches).",
  },

  hypothesis:
    "Freelancers will trust an AI-drafted fixed-scope contract enough to want it sent to them — the friction isn't drafting prose, it's knowing what clauses protect them.",
  successMetric: "poc_signal / poc_visit >= 8%",
  killCondition: "<3% signal after 300 visits or 14 days, whichever first.",
};
