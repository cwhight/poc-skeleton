/**
 * Worked example — ReviewPilot.
 * Manager pastes rough notes about a report → gets a structured performance review.
 * Riskiest assumption: do managers trust an AI to turn private notes into review prose they'd submit?
 */

import type { IdeaConfig } from "../idea.config.example";

export const ideaConfig: IdeaConfig = {
  name: "ReviewPilot",
  slug: "reviewpilot",
  tagline: "Turn your scattered notes into a fair, structured performance review.",

  landingHeadline: "Paste your notes. Get a performance review you'd be happy to deliver.",
  subcopy:
    "For managers staring at review season with a folder of half-formed notes. ReviewPilot turns bullet points and half-remembered moments into a balanced review — strengths, growth areas, and concrete examples — in the tone you'd actually use.",

  inputLabel: "Paste your notes on this person (wins, misses, examples, the period)",
  placeholder:
    "Sam, Q1-Q2. Shipped the billing migration ahead of schedule. Great in incidents. Quiet in planning, rarely pushes back. Mentored two juniors. Missed the docs deadline twice.",
  maxInputBytes: 16_000,

  systemPrompt: [
    "You are an experienced people manager helping write a fair, evidence-based performance review.",
    "Turn the user's rough notes into a structured review with sections: Summary, Key Strengths (with the specific examples from the notes), Areas for Growth (constructive, forward-looking), Notable Contributions, and Suggested Focus for Next Period.",
    "Ground every claim in something the user actually wrote — never invent achievements, metrics, or incidents. If the notes are thin in a section, say what's missing rather than padding.",
    "Balanced and specific: pair growth areas with a concrete next step. Professional, warm, direct — no corporate filler.",
    "Use second-person reference to the employee by the name given (or [EMPLOYEE] if none).",
    "Output GitHub-flavoured Markdown only. No commentary before or after the review.",
  ].join("\n"),

  outputFormat: "markdown",
  ctaText: "Copy review",

  signalQuestion: "Would you use this as the basis for a real performance review?",
  signal: {
    kind: "email",
    consentCopy: "Email me this review (and tell me when ReviewPilot launches).",
  },

  hypothesis:
    "Managers will trust AI to structure their own raw notes into review prose — the pain is the blank page and balance, not the writing itself — enough to leave an email.",
  successMetric: "poc_signal / poc_visit >= 10%",
  killCondition: "<4% signal after 250 visits or 14 days, whichever first.",
};
