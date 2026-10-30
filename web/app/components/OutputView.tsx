"use client";

/**
 * OutputView — the SINGLE render path for generated (LLM) output.
 *
 * SECURITY (a hard, learned requirement — see PATTERNS-BAKED-IN):
 *   HTML-escape the model output FIRST, then render markdown over the escaped
 *   string, layering only OUR OWN tags. We never feed raw LLM text to
 *   dangerouslySetInnerHTML. Concretely:
 *     1. escapeHtml(text)  -> &lt; &gt; &amp; &quot; &#39; so any tags/scripts
 *        the model emitted become inert literal characters.
 *     2. marked.parse(escaped) with no HTML pass-through, so markdown adds only
 *        the tags marked itself generates (<h2>, <strong>, <pre>, ...). Because
 *        the input was already escaped, marked cannot resurrect raw HTML.
 *   Plaintext mode skips markdown entirely: escaped text in a <pre>.
 *
 * This component is used everywhere generated content is shown so no surface
 * can diverge from the safe path.
 */

import { useMemo, useState, useCallback } from "react";
import { Copy, Check, Download } from "lucide-react";
import { marked } from "marked";

export function escapeHtml(input: string): string {
  return input
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

// Configure marked to be deterministic and to NOT emit/parse raw HTML.
// (Input is pre-escaped, so this is belt-and-braces.)
marked.setOptions({
  gfm: true,
  breaks: false,
});

function renderSafeMarkdown(text: string): string {
  const escaped = escapeHtml(text);
  // marked.parse is synchronous here (no async extensions configured).
  const html = marked.parse(escaped, { async: false }) as string;
  return html;
}

export function OutputView({
  text,
  format,
  ctaText,
  onAction,
}: {
  text: string;
  format: "markdown" | "plaintext";
  /** Primary action label from idea.config (e.g. "Copy contract"). */
  ctaText?: string;
  /** Fired on copy/download so the page can emit poc_output_action. */
  onAction?: (action: "copy" | "download") => void;
}) {
  const [copied, setCopied] = useState(false);

  const html = useMemo(() => {
    if (format === "markdown") return renderSafeMarkdown(text);
    // plaintext: escaped, preserved whitespace.
    return `<pre class="ov-plain">${escapeHtml(text)}</pre>`;
  }, [text, format]);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      // ignore — clipboard may be unavailable
    }
    onAction?.("copy");
  }, [text, onAction]);

  const handleDownload = useCallback(() => {
    const ext = format === "markdown" ? "md" : "txt";
    const blob = new Blob([text], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `output.${ext}`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
    onAction?.("download");
  }, [text, format, onAction]);

  const copyLabel = ctaText ?? "Copy";

  return (
    <div className="output">
      <div className="output-actions">
        <button type="button" className="btn btn-primary" onClick={handleCopy} aria-live="polite">
          {copied ? <Check size={18} aria-hidden /> : <Copy size={18} aria-hidden />}
          <span>{copied ? "Copied" : copyLabel}</span>
        </button>
        <button type="button" className="btn btn-secondary" onClick={handleDownload}>
          <Download size={18} aria-hidden />
          <span>Download</span>
        </button>
      </div>
      {/* Safe: `html` is escaped-then-markdown — only our own tags, never raw LLM HTML. */}
      <article className="output-body" dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  );
}
