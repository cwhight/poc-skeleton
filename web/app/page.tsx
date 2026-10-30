"use client";

/**
 * Landing page. Renders the hero from idea.config (tagline / landingHeadline /
 * subcopy) and a single CTA to /generate. Emits `poc_visit` on mount — funnel
 * step 1, by construction. Honest copy only (whatever the config says — no
 * invented metrics added here). Vendor-neutral (never names the model).
 */

import { useEffect } from "react";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { ideaConfig } from "../idea.config";
import { track } from "./components/analytics";

export default function LandingPage() {
  useEffect(() => {
    track("poc_visit");
  }, []);

  return (
    <main>
      <section className="hero">
        <span className="tagline">{ideaConfig.tagline}</span>
        <h1>{ideaConfig.landingHeadline}</h1>
        <p>{ideaConfig.subcopy}</p>
        <Link href="/generate" className="btn btn-primary hero-cta">
          <span>Try it</span>
          <ArrowRight size={18} aria-hidden />
        </Link>
      </section>
    </main>
  );
}
