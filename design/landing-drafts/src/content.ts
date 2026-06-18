// Canonical Qratum content. All five drafts import this so the comparison is
// about DESIGN, not messaging. Do not fork wording per draft — adapt voice only.

export type RoadmapStatus = "done" | "now" | "next" | "later";

export const qratum = {
  name: "Qratum",
  wordmark: "qratum",
  tagline: "The local librarian for your AI coding sessions.",
  oneLiner:
    "Qratum is a local-first library, vault, and refinery for your AI coding sessions. It captures, preserves, normalizes, redacts, reviews, and verifies your Claude Code sessions — without ever uploading your raw transcripts.",

  problem: {
    headline: "Your AI coding sessions are ephemeral.",
    body:
      "Claude Code purges transcripts after roughly 30 days. Months of debugging, decisions, and trajectories vanish — and nothing records where that data came from. Qratum fixes both: it preserves, and it remembers provenance.",
  },

  pillars: [
    {
      key: "vault",
      name: "The Vault",
      summary:
        "Capture into a content-addressed archive. A transcript the tool deletes tomorrow is recoverable.",
      points: [
        "Copy-on-capture: every session end writes a blob plus a ref",
        "Dedup by sha256 digest — backfill is a no-op the second time",
        "Backups that prove themselves: `qrt vault backup --verify`",
        "Survives abandonment gracefully — passive value from day one",
      ],
    },
    {
      key: "refinery",
      name: "The Refinery",
      summary:
        "On-demand: normalize → redact → evidence → review → report → corpus. It runs only when you ask.",
      points: [
        "Deterministic redaction as the first safety boundary",
        "Evidence signals: edits-after-verify, no final test, retry loops",
        "No daemon, no queue, no automatic review",
        "ADP and native corpus JSONL export, eligibility-checked",
      ],
    },
    {
      key: "provenance",
      name: "Provenance",
      summary:
        "The system of record for where session data came from — raw history, digests, derivations.",
      points: [
        "Every object carries schema_version, producer, transform version",
        "Data-class lineage: raw → redacted → review → corpus → published",
        "Content-addressed blobs are immutable; refs are the pointers",
        "Tombstones, never silent deletion",
      ],
    },
  ],

  loop: [
    { step: "Capture", detail: "A global SessionEnd hook writes one event and copies the transcript into the vault." },
    { step: "Archive safely", detail: "Content-addressed blob store, deduped by digest, local-only by default." },
    { step: "Review", detail: "Deterministic signals become findings inside a review envelope." },
    { step: "Local library", detail: "Sessions and metrics, listed locally for reuse." },
    { step: "Corpus candidate", detail: "Eligibility-checked export — only on an explicit command." },
  ],

  trust: {
    headline: "Local-first is not a feature. It is the architecture.",
    items: [
      { title: "Raw never leaves", body: "Raw transcripts stay on your machine unless you explicitly approve. There is no cloud by default." },
      { title: "Trust boundaries", body: "No boundary may silently upgrade to a more sensitive data class. Raw to external or published always needs explicit consent." },
      { title: "Redaction gates export", body: "Deterministic redaction is the first downgrade boundary. Redaction uncertainty blocks export until you decide." },
      { title: "No raw routes, ever", body: "The first app shell has no raw transcript APIs. No raw content in logs, spans, or metrics — paths and digests only." },
    ],
  },

  roadmap: [
    { phase: "Milestone A", status: "done" as RoadmapStatus, title: "Vertical slice", detail: "Capture, spool, deterministic redaction, evidence, review cards, reports, ADP export." },
    { phase: "P0", status: "done" as RoadmapStatus, title: "Spec & contracts", detail: "Schema registry, JSON Schemas, ADRs, fixtures." },
    { phase: "P1", status: "done" as RoadmapStatus, title: "The Vault", detail: "Global capture, content-addressed raw archive, dedup, `vault doctor`, `backup --verify`." },
    { phase: "P2", status: "done" as RoadmapStatus, title: "Verification trust gate", detail: "`qrt trust` scorecard, reflection-canary redaction proofs, `vault gc`/`erase`/`install-schedule`, data_class lineage, schema conformance." },
    { phase: "v0.1.0", status: "now" as RoadmapStatus, title: "First release", detail: "`brew install qratum` — single Go binary, darwin + linux." },
    { phase: "Next", status: "next" as RoadmapStatus, title: "Cross-repo curation", detail: "Consumer-side memory-import receipts, gated on the personal-memory gateway producer." },
    { phase: "Later", status: "later" as RoadmapStatus, title: "Hardening", detail: "PII/third-party detection, audit-log tamper-evidence, multi-machine merge — named, deferred future work." },
  ],

  stats: [
    { value: "1", label: "Go binary · `qrt`" },
    { value: "0", label: "Cloud dependencies" },
    { value: "0", label: "Accounts required" },
    { value: "30d", label: "Sessions Claude would delete" },
  ],

  cta: {
    primary: { label: "Read the spec", href: "/SPEC.md" },
    secondary: { label: "brew install qratum", href: "#install" },
  },

  install: [
    "brew tap edictum-ai/edictum",
    "brew install qratum    # the qrt binary",
    "qrt hook install       # capture every future session",
    "qrt trust              # the self-verification scorecard",
  ],

  honest: [
    "Redaction is credentials-only and best-effort alpha; PII and third-party content are preserved verbatim. The trust gate names the residual classes that can still leak.",
    "Capture and refine are Claude-Code-only. Cloud and web sessions are not captured.",
    "Not a knowledge store, not an insights generator, not a publisher between your own tools.",
  ],

  identity: [
    "local-first", "single binary", "content-addressed", "deterministic",
    "provenance", "trust boundaries", "consent-gated", "no telemetry",
  ],

  ecosystem: [
    { name: "Qratum", role: "the librarian: vault + refinery for AI session data" },
    { name: "Edictum", role: "runtime process enforcement for AI agents" },
    { name: "Ductum", role: "the AI Software Factory" },
  ],
};

export type QratumContent = typeof qratum;
