// Canonical Qratum content. All five drafts import this so the comparison is
// about DESIGN, not messaging. Do not fork wording per draft — adapt voice only.

export type RoadmapStatus = "done" | "now" | "next" | "later";

export const qratum = {
  name: "Qratum",
  wordmark: "qratum",
  tagline: "The local librarian for your AI coding sessions.",
  oneLiner:
    "Qratum is a local-first library, vault, and review pipeline for your AI coding sessions. It captures, preserves, normalizes, redacts, reviews, and searches every session — without ever uploading your raw transcripts.",

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
    { step: "Searchable library", detail: "Sessions, revisions, and metrics, indexed locally for reuse." },
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
    { phase: "Milestone A", status: "done" as RoadmapStatus, title: "Vertical slice proven", detail: "Capture, spool, deterministic redaction, evidence, review cards, reports, ADP export." },
    { phase: "P0", status: "now" as RoadmapStatus, title: "Spec & contracts", detail: "Locking the redesign — schema registry, JSON Schemas, fixtures — before any runtime code." },
    { phase: "P1", status: "next" as RoadmapStatus, title: "The Vault", detail: "Global capture, content-addressed raw archive, `vault doctor`, `backup --verify`." },
    { phase: "P2", status: "later" as RoadmapStatus, title: "Import & revisions", detail: "`import --all` wizard, session revisions, jobs, individual retry." },
    { phase: "P3", status: "later" as RoadmapStatus, title: "Review + local app", detail: "Review envelope, local app shell on 127.0.0.1:9218, report v2." },
    { phase: "P4", status: "later" as RoadmapStatus, title: "Lessons & insights", detail: "Local and external AI providers behind consent; cross-session insights." },
    { phase: "P5", status: "later" as RoadmapStatus, title: "Corpus & publish", detail: "Native corpus JSONL, ADP export, local-folder publishing, manual approval." },
  ],

  stats: [
    { value: "1", label: "Go binary · `qrt`" },
    { value: "0", label: "Cloud dependencies" },
    { value: "0", label: "Accounts required" },
    { value: "30d", label: "Sessions Claude would delete" },
  ],

  cta: {
    primary: { label: "Read the spec", href: "/SPEC.md" },
    secondary: { label: "make build", href: "#install" },
  },

  install: [
    "make build",
    "qrt hook install       # global SessionEnd capture",
    "qrt vault doctor       # is preservation actually working?",
    "qrt vault backfill     # inventory existing transcripts",
  ],

  honest: [
    "Deterministic redaction is best-effort alpha.",
    "Status: pre-1.0 spec phase (P0). The vault-first proposal is under review.",
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
