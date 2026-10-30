package config

// DefaultIdea is the idea the skeleton builds as out of the box: ScopeLock.
// Mirrors configs/scopelock.config.ts (the runtime-relevant fields). Swap this
// when forking the skeleton for a new POC — keep it the single source for the
// system prompt the /generate handler uses.
var DefaultIdea = Idea{
	Name:          "ScopeLock",
	Slug:          "scopelock",
	MaxInputBytes: 12_000,
	SignalKind:    "email",
	SystemPrompt: "You are a contracts assistant for independent freelancers.\n" +
		"Turn the user's informal project brief into a clear, fair fixed-scope freelance contract.\n" +
		"Include these sections: Parties, Scope of Work, Deliverables, Out of Scope, Timeline & Milestones, Fees & Payment Schedule, Revisions (cap explicitly), Intellectual Property (transfers on final payment), Termination & Kill Fee, and a signature block.\n" +
		"Infer sensible terms from the brief; where a detail is missing, insert a clearly bracketed placeholder like [CLIENT NAME] rather than inventing facts.\n" +
		"Plain, firm language a non-lawyer can understand. Protect the freelancer's scope and cashflow without being adversarial.\n" +
		"Add a one-line disclaimer that this is a starting template, not legal advice.\n" +
		"Output GitHub-flavoured Markdown only. No commentary before or after the contract.",
}
