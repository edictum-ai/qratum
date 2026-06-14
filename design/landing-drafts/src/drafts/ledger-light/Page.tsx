import { useState } from "react";
import { motion } from "framer-motion";
import type { Variants } from "framer-motion";
import { qratum, type RoadmapStatus } from "../../content";
import "./style.css";
import Terminal from "./Terminal";
import Typewriter from "./Typewriter";
import CountUp from "./CountUp";
import {
  VaultIcon,
  RefineryIcon,
  ProvenanceIcon,
  ShieldIcon,
  TerminalIcon,
  LockIcon,
  RouteOffIcon,
  ArrowRight,
  CheckIcon,
  ClockIcon,
  TrashIcon,
  CopyIcon,
} from "./icons";

/* ----------------------------------------------------------------
   Dark Ledger — Neobrutalism on OLED dark.
   Motion = librarian: long, quiet fades (~0.8s) on a calm curve
   [0.22,1,0.36,1], refined slow stagger. No punchy springs as the
   default. Microcopy = refinery (crisp technical labels).
----------------------------------------------------------------- */

const easeOut = [0.22, 1, 0.36, 1] as const;

const reveal: Variants = {
  hidden: { opacity: 0, y: 22 },
  show: { opacity: 1, y: 0, transition: { duration: 0.8, ease: easeOut } },
};

const stagger: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.1, delayChildren: 0.05 } },
};

const staggerSlow: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.14, delayChildren: 0.08 } },
};

const VIEWPORT = { once: true, amount: 0.25 } as const;

/* ---- inline line icons config ---- */
const iconStroke = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 2,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
};

/* ---- data for the manifest / marquee motifs (illustrative) ---- */
type ManifestRow = {
  kind: string;
  chip: "raw" | "redacted" | "review" | "metrics" | "lesson" | "corpus";
  digest: string;
  size: string;
  status: { label: string; tone: "ok" | "blocked" | "pending" };
};

const MANIFEST_ROWS: ManifestRow[] = [
  { kind: "session.jsonl", chip: "raw", digest: "sha256:7f3a…b21e", size: "418 KB", status: { label: "local", tone: "ok" } },
  { kind: "transcript.redacted.jsonl", chip: "redacted", digest: "sha256:9c02…eef4", size: "302 KB", status: { label: "verified", tone: "ok" } },
  { kind: "review-envelope.json", chip: "review", digest: "sha256:1b88…44ac", size: "11 KB", status: { label: "findings: 3", tone: "pending" } },
  { kind: "metrics.json", chip: "metrics", digest: "sha256:5d6e…0a91", size: "4 KB", status: { label: "indexed", tone: "ok" } },
  { kind: "corpus.jsonl", chip: "corpus", digest: "sha256:aa01…ff3d", size: "—", status: { label: "blocked", tone: "blocked" } },
];

const TICKER = [
  "sha256:7f3a…b21e",
  "sha256:9c02…eef4",
  "sha256:1b88…44ac",
  "sha256:5d6e…0a91",
  "sha256:aa01…ff3d",
  "sha256:c774…92be",
  "sha256:3e0d…7710",
];

const ROADMAP_TONE: Record<RoadmapStatus, string> = {
  done: "dl-rm-done",
  now: "dl-rm-now",
  next: "dl-rm-next",
  later: "dl-rm-later",
};

const ROADMAP_LABEL: Record<RoadmapStatus, string> = {
  done: "Done",
  now: "Now",
  next: "Next",
  later: "Later",
};

const PILLAR_ICON = {
  vault: VaultIcon,
  refinery: RefineryIcon,
  provenance: ProvenanceIcon,
} as const;

const TRUST_ICONS = [LockIcon, ShieldIcon, RouteOffIcon, RouteOffIcon];

const digest = (s: string) =>
  "sha256:" + s.toLowerCase().replace(/[^a-z0-9]/g, "").slice(0, 4) + "…" + s.slice(-4);

/* ---------------------------------------------------------------- */
/* Sub-components                                                    */
/* ---------------------------------------------------------------- */

function Marquee() {
  const items = [...TICKER, ...TICKER];
  return (
    <div className="dl-marquee" aria-hidden="true">
      <div className="dl-marquee-track">
        {items.map((d, i) => (
          <span className="dl-marquee-item" key={i}>
            <span className="dl-marquee-tag">digest</span>
            {d}
            <span className="dl-marquee-sep">◆</span>
          </span>
        ))}
      </div>
    </div>
  );
}

function ManifestTable() {
  return (
    <div className="dl-manifest" role="figure" aria-label="Example vault manifest">
      <div className="dl-manifest-head">
        <span className="dl-label">vault.manifest</span>
        <span className="dl-manifest-dots" aria-hidden="true">
          <span />
          <span />
          <span />
        </span>
      </div>
      <table className="dl-manifest-table">
        <thead>
          <tr>
            <th>kind</th>
            <th>class</th>
            <th>digest</th>
            <th>size</th>
            <th>status</th>
          </tr>
        </thead>
        <tbody>
          {MANIFEST_ROWS.map((r) => (
            <tr key={r.digest}>
              <td>{r.kind}</td>
              <td>
                <span className={`dl-chip dl-chip-${r.chip}`}>{r.chip}</span>
              </td>
              <td className="digest">{r.digest}</td>
              <td>{r.size}</td>
              <td>
                <span className={`dl-status dl-status-${r.status.tone}`}>
                  {r.status.label}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/* ---------------------------------------------------------------- */
/* Page                                                              */
/* ---------------------------------------------------------------- */

export default function DarkLedgerPage() {
  return (
    <div className="ledger-light-root">
      {/* ---------------- NAV ---------------- */}
      <nav className="dl-nav" aria-label="Primary">
        <div className="dl-wrap dl-nav-inner">
          <a className="dl-wordmark" href="#top" aria-label="qratum home">
            {qratum.wordmark}
          </a>
          <div className="dl-nav-links">
            <a className="dl-nav-link" href="#pillars">Vault</a>
            <a className="dl-nav-link" href="#roadmap">Roadmap</a>
            <a className="dl-nav-link" href="#install">Install</a>
            <a
              className="dl-nav-link dl-nav-cta"
              href={qratum.cta.primary.href}
            >
              {qratum.cta.primary.label} →
            </a>
          </div>
        </div>
      </nav>

      <main id="top">
        {/* ---------------- HERO ---------------- */}
        <motion.section
          className="dl-hero"
          initial="hidden"
          animate="show"
          variants={staggerSlow}
        >
          <div className="dl-wrap dl-hero-grid">
            <div>
              <motion.div variants={reveal}>
                <span className="dl-eyebrow dl-label">
                  <span className="dl-dot" aria-hidden="true" />
                  local-first · content-addressed · single binary
                </span>
              </motion.div>
              <motion.h1 className="dl-display" variants={reveal}>
                The local <span className="dl-mark-yellow">librarian</span> for your AI coding sessions.
              </motion.h1>
              <motion.div className="dl-hero-lead" variants={reveal}>
                <span className="dl-hero-prompt">qrt&gt;&nbsp;</span>
                <Typewriter text={qratum.oneLiner} className="dl-mono" />
              </motion.div>
              <motion.p className="dl-hero-hook" variants={reveal}>
                <span className="dl-warn">⚠</span> {qratum.problem.headline}{" "}
                {qratum.problem.body}
              </motion.p>
              <motion.div className="dl-cta-row" variants={reveal}>
                <a className="dl-btn dl-btn-primary" href={qratum.cta.primary.href}>
                  <TerminalIcon width={15} height={15} />
                  {qratum.cta.primary.label}
                </a>
                <a className="dl-btn dl-btn-ghost" href={qratum.cta.secondary.href}>
                  {qratum.cta.secondary.label}
                  <ArrowRight width={15} height={15} />
                </a>
              </motion.div>
              <motion.div className="dl-digest-row" variants={reveal}>
                <span className="dl-chip dl-chip-digest">{digest("qrt-binary")}</span>
                <span className="dl-chip">go · content-addressed</span>
                <span className="dl-chip">no telemetry</span>
              </motion.div>
            </div>
            <motion.div variants={reveal}>
              <Terminal />
            </motion.div>
          </div>
        </motion.section>

        {/* ---------------- MARQUEE ---------------- */}
        <Marquee />

        {/* ---------------- PROBLEM ---------------- */}
        <section className="dl-section" aria-labelledby="problem-title">
          <div className="dl-wrap">
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.span className="dl-eyebrow dl-label" variants={reveal}>
                <span className="dl-dot" aria-hidden="true" />
                01 · refinery · the problem
              </motion.span>
              <motion.h2
                id="problem-title"
                className="dl-display dl-section-title"
                variants={reveal}
              >
                Your sessions are <span className="dl-mark-cyan">ephemeral.</span>
              </motion.h2>
            </motion.div>

            <motion.div
              className="dl-problem"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.div className="dl-problem-card" variants={reveal}>
                <span className="dl-label" style={{ color: "rgba(255,255,255,0.82)" }}>
                  purge timer
                </span>
                <div
                  className="dl-display"
                  style={{ fontSize: "clamp(2.8rem,7vw,5rem)", margin: "10px 0 8px" }}
                >
                  ~30d
                </div>
                <p style={{ margin: 0, fontSize: "0.98rem", lineHeight: 1.5 }}>
                  {qratum.problem.body}
                </p>
              </motion.div>

              <div className="dl-problem-points">
                {[
                  { icon: <ClockIcon width={18} height={18} />, stat: "~30d", title: "Sessions get purged", body: "Claude Code deletes transcripts after roughly 30 days." },
                  { icon: <TrashIcon width={18} height={18} />, stat: "∞", title: "Months of work vanish", body: "Debugging, decisions, and trajectories gone for good." },
                  { icon: <RouteOffIcon width={18} height={18} />, stat: "?", title: "No provenance recorded", body: "Nothing tracks where the data came from in the first place." },
                ].map((c, i) => (
                  <motion.div className="dl-problem-point" key={i} variants={reveal}>
                    <span className="dl-problem-num">{String(i + 1).padStart(2, "0")}</span>
                    <div>
                      <div
                        className="dl-mono"
                        style={{ fontSize: "0.92rem", color: "var(--dl-ink)", marginBottom: 4, letterSpacing: "0.04em" }}
                      >
                        {c.stat} · {c.title}
                      </div>
                      <span style={{ fontSize: "0.9rem", lineHeight: 1.45, color: "var(--dl-ink-soft)" }}>
                        {c.body}
                      </span>
                    </div>
                  </motion.div>
                ))}
              </div>
            </motion.div>
          </div>
        </section>

        {/* ---------------- PILLARS ---------------- */}
        <section className="dl-section" id="pillars" aria-labelledby="pillars-title">
          <div className="dl-wrap">
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.span className="dl-eyebrow dl-label" variants={reveal}>
                <span className="dl-dot" aria-hidden="true" />
                02 · three pillars
              </motion.span>
              <motion.h2
                id="pillars-title"
                className="dl-display dl-section-title"
                variants={reveal}
              >
                Vault. Refinery. Provenance.
              </motion.h2>
              <motion.p className="dl-section-intro" variants={reveal}>
                Preserve what the tools delete. Refine on demand. Always know where
                the data came from.
              </motion.p>
            </motion.div>

            <motion.div
              className="dl-pillars"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={staggerSlow}
            >
              {qratum.pillars.map((pillar) => {
                const Icon = PILLAR_ICON[pillar.key as keyof typeof PILLAR_ICON] ?? VaultIcon;
                return (
                  <motion.article key={pillar.key} className="dl-pillar" variants={reveal}>
                    <div className="dl-pillar-head">
                      <span className="dl-chip dl-chip-metrics">{pillar.key}</span>
                      <span className="dl-pillar-icon-box" aria-hidden="true">
                        <Icon width={20} height={20} />
                      </span>
                    </div>
                    <div className="dl-pillar-body">
                      <div className="dl-pillar-name">{pillar.name}</div>
                      <p className="dl-pillar-summary">{pillar.summary}</p>
                      <ul className="dl-pillar-points">
                        {pillar.points.map((pt, i) => (
                          <li key={i}>
                            <CheckIcon className="dl-pillar-check" width={14} height={14} {...iconStroke} />
                            <span>{pt}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  </motion.article>
                );
              })}
            </motion.div>
          </div>
        </section>

        {/* ---------------- HOW IT WORKS ---------------- */}
        <section className="dl-section" aria-labelledby="loop-title">
          <div className="dl-wrap">
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.span className="dl-eyebrow dl-label" variants={reveal}>
                <span className="dl-dot" aria-hidden="true" />
                03 · capture → archive → review
              </motion.span>
              <motion.h2
                id="loop-title"
                className="dl-display dl-section-title"
                variants={reveal}
              >
                normalize → redact → evidence → review → corpus
              </motion.h2>
              <motion.p className="dl-section-intro" variants={reveal}>
                On-demand by design. The vault passively captures on every session
                end; everything else runs only when you ask.
              </motion.p>
            </motion.div>

            <motion.div
              className="dl-loop"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
              aria-label="Qratum session loop"
            >
              {qratum.loop.map((step, i) => {
                const tones = ["yellow", "", "", "", "cyan"] as const;
                const tone = tones[i] ?? "";
                return (
                  <motion.div
                    key={step.step}
                    className="dl-loop-step"
                    data-tone={tone}
                    variants={reveal}
                  >
                    <span className="dl-loop-num">0{i + 1}</span>
                    <span className="dl-loop-name">{step.step}</span>
                    <span className="dl-loop-detail">{step.detail}</span>
                    {i < qratum.loop.length - 1 && (
                      <span className="dl-loop-digest">{digest("step-" + (i + 1))} →</span>
                    )}
                  </motion.div>
                );
              })}
            </motion.div>

            {/* manifest table — the hero artifact of the ops feel */}
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
              style={{ marginTop: 36 }}
            >
              <motion.div variants={reveal}>
                <ManifestTable />
              </motion.div>
            </motion.div>
          </div>
        </section>

        {/* ---------------- TRUST ---------------- */}
        <section className="dl-section" aria-labelledby="trust-title">
          <div className="dl-wrap">
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.span className="dl-eyebrow dl-label" variants={reveal}>
                <span className="dl-dot" aria-hidden="true" />
                04 · trust &amp; security · consent-gated
              </motion.span>
              <motion.h2
                id="trust-title"
                className="dl-display dl-section-title"
                variants={reveal}
              >
                {qratum.trust.headline}
              </motion.h2>
            </motion.div>

            <motion.div
              className="dl-trust"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              {qratum.trust.items.map((item, i) => {
                const Icon = TRUST_ICONS[i] ?? ShieldIcon;
                return (
                  <motion.div className="dl-trust-item" key={i} variants={reveal}>
                    <span className="dl-trust-icon" aria-hidden="true">
                      <Icon width={20} height={20} />
                    </span>
                    <div>
                      <h3 className="dl-trust-title">{item.title}</h3>
                      <p className="dl-trust-body">{item.body}</p>
                    </div>
                  </motion.div>
                );
              })}
            </motion.div>
          </div>
        </section>

        {/* ---------------- STATS ---------------- */}
        <section className="dl-section" aria-label="Key figures">
          <div className="dl-wrap">
            <motion.div
              className="dl-stats"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              {qratum.stats.map((s, i) => {
                const tones = ["", "cyan", "", "green"] as const;
                const tone = tones[i] ?? "";
                return (
                  <motion.div className="dl-stat" key={s.label} variants={reveal}>
                    <span className="dl-stat-value" data-tone={tone}>
                      <CountUp value={s.value} />
                    </span>
                    <span className="dl-stat-label">{s.label}</span>
                  </motion.div>
                );
              })}
            </motion.div>
          </div>
        </section>

        {/* ---------------- ROADMAP ---------------- */}
        <section className="dl-section" id="roadmap" aria-labelledby="roadmap-title">
          <div className="dl-wrap">
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.span className="dl-eyebrow dl-label" variants={reveal}>
                <span className="dl-dot" aria-hidden="true" />
                05 · roadmap · honest status
              </motion.span>
              <motion.h2
                id="roadmap-title"
                className="dl-display dl-section-title"
                variants={reveal}
              >
                Pre-1.0. Spec phase. <span className="dl-mark-cyan">Honest.</span>
              </motion.h2>
              <motion.p className="dl-section-intro" variants={reveal}>
                Status: pre-1.0 spec phase (P0). Milestone A is the only thing proven
                so far. The vault-first proposal is under review — nothing here is
                promised as shipped.
              </motion.p>
            </motion.div>

            <motion.div
              className="dl-roadmap"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
              aria-label="Qratum roadmap"
            >
              {qratum.roadmap.map((r) => (
                <motion.div
                  className="dl-roadmap-row"
                  key={r.phase}
                  data-dim={r.status === "later"}
                  variants={reveal}
                >
                  <div className="dl-roadmap-cell">
                    <span className={`dl-rm-badge ${ROADMAP_TONE[r.status]}`}>
                      {r.status === "done" && <CheckIcon width={11} height={11} />}
                      {ROADMAP_LABEL[r.status]}
                    </span>
                  </div>
                  <div className="dl-roadmap-cell">
                    <span className="dl-roadmap-phase">{r.phase}</span>
                  </div>
                  <div className="dl-roadmap-cell">
                    <span className="dl-roadmap-detail">
                      <strong style={{ fontWeight: 700 }}>{r.title}.</strong>{" "}
                      {r.detail}
                    </span>
                  </div>
                </motion.div>
              ))}
            </motion.div>
          </div>
        </section>

        {/* ---------------- INSTALL / CTA ---------------- */}
        <section className="dl-section" id="install" aria-labelledby="install-title">
          <div className="dl-wrap">
            <motion.div
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.span className="dl-eyebrow dl-label" variants={reveal}>
                <span className="dl-dot" aria-hidden="true" />
                06 · install · eligibility-checked
              </motion.span>
              <motion.h2
                id="install-title"
                className="dl-display dl-section-title"
                variants={reveal}
              >
                One binary. <span className="dl-mark-yellow">make build.</span>
              </motion.h2>
            </motion.div>

            <motion.div
              className="dl-install-grid"
              initial="hidden"
              whileInView="show"
              viewport={VIEWPORT}
              variants={stagger}
            >
              <motion.div variants={reveal}>
                <InstallTerminal />
              </motion.div>

              <motion.div className="dl-install-aside" variants={reveal}>
                <h3>Preserve what the tool will delete.</h3>
                <p>
                  A single Go binary. No accounts, no cloud, no telemetry. Install
                  the global SessionEnd hook and the vault begins recording on its
                  own — raw never leaves your machine.
                </p>
                <div className="dl-cta-row">
                  <a className="dl-btn dl-btn-primary" href={qratum.cta.primary.href}>
                    <TerminalIcon width={15} height={15} />
                    {qratum.cta.primary.label}
                  </a>
                  <a className="dl-btn dl-btn-ghost" href="#top">
                    Back to top
                  </a>
                </div>
              </motion.div>
            </motion.div>
          </div>
        </section>
      </main>

      {/* ---------------- FOOTER ---------------- */}
      <footer className="dl-footer" aria-label={`${qratum.name} site footer`}>
        <div className="dl-wrap">
          {/* honesty lines */}
          <div className="dl-footer-honest">
            <span className="dl-label" style={{ color: "var(--dl-clay)", marginBottom: 14, display: "block" }}>
              honest status
            </span>
            <ul>
              {qratum.honest.map((line, i) => (
                <li key={i}>{line}</li>
              ))}
            </ul>
          </div>

          <div className="dl-footer-cols">
            <div className="dl-footer-brand">
              <span className="dl-wordmark">{qratum.wordmark}</span>
              <p className="dl-footer-tagline">{qratum.tagline}</p>
              <div className="dl-identity" style={{ marginTop: 18 }}>
                {qratum.identity.map((id) => (
                  <span key={id} className="dl-identity-chip">
                    {id}
                  </span>
                ))}
              </div>
            </div>

            <div>
              <h3 className="dl-footer-col-title">Ecosystem</h3>
              <ul className="dl-footer-ecosystem">
                {qratum.ecosystem.map((e) => {
                  const internal = e.name === "Ductum" || e.name === "personal-memory";
                  return (
                    <li key={e.name}>
                      <span className="dl-footer-eco-name" data-dim={internal || undefined}>
                        {e.name}
                        {internal && <span className="dl-footer-eco-tag">internal</span>}
                      </span>
                      <span className="dl-footer-eco-role">{e.role}</span>
                    </li>
                  );
                })}
              </ul>
            </div>

            <div>
              <h3 className="dl-footer-col-title">Navigate</h3>
              <ul className="dl-footer-ecosystem">
                <li>
                  <a className="dl-footer-eco-name is-link" href="#pillars">
                    Vault
                  </a>
                  <span className="dl-footer-eco-role">Content-addressed archive</span>
                </li>
                <li>
                  <a className="dl-footer-eco-name is-link" href="#roadmap">
                    Roadmap
                  </a>
                  <span className="dl-footer-eco-role">Pre-1.0 spec phase</span>
                </li>
                <li>
                  <a className="dl-footer-eco-name is-link" href={qratum.cta.primary.href}>
                    Spec
                  </a>
                  <span className="dl-footer-eco-role">{qratum.cta.primary.href}</span>
                </li>
              </ul>
            </div>
          </div>

          <div className="dl-footer-bottom">
            <span className="dl-footer-copy">
              © {new Date().getFullYear()} {qratum.name} · local-first, single binary
            </span>
            <span className="dl-footer-fineprint">
              Design draft · Dark Ledger (Neobrutalism on OLED) — not part of qratum runtime or CI
            </span>
          </div>
        </div>
      </footer>
    </div>
  );
}

/* ---------------------------------------------------------------- */
/* Install terminal with copy-to-clipboard                          */
/* ---------------------------------------------------------------- */

function InstallTerminal() {
  const [copied, setCopied] = useState(false);

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(qratum.install.join("\n"));
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard unavailable */
    }
  };

  return (
    <div className="dl-terminal">
      <div className="dl-terminal-bar">
        <span className="dl-term-dot" style={{ background: "#d6453d" }} />
        <span className="dl-term-dot" style={{ background: "#f5d90a" }} />
        <span className="dl-term-dot" style={{ background: "#3fdd8a" }} />
        <span className="dl-mono dl-terminal-label">setup.sh</span>
        <span style={{ flex: 1 }} />
        <button
          onClick={onCopy}
          aria-label="Copy install commands"
          className="dl-copy-btn"
        >
          {copied ? <CheckIcon width={12} height={12} /> : <CopyIcon width={12} height={12} />}
          {copied ? "copied" : "copy"}
        </button>
      </div>
      <div className="dl-terminal-body">
        {qratum.install.map((line, i) => {
          const comment = line.includes("#");
          return (
            <div key={i} style={{ display: "flex", gap: 9 }}>
              <span style={{ color: "var(--dl-yellow)", flexShrink: 0 }}>
                {comment ? "#" : "$"}
              </span>
              {comment ? (
                <span className="dl-cm-comment">{line.replace(/^.*#\s*/, "# ")}</span>
              ) : (
                <CmdSpans line={line} />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

/** Light syntax highlighting for install command lines. */
function CmdSpans({ line }: { line: string }) {
  const parts = line.trim().split(/(\s+)/);
  return (
    <>
      {parts.map((p, idx) => {
        if (/^\s+$/.test(p)) return <span key={idx}>{p}</span>;
        if (idx === 0) return <span key={idx} className="dl-cm-cmd">{p}</span>;
        if (p.startsWith("-") || p.startsWith("--")) return <span key={idx} className="dl-cm-arg">{p}</span>;
        return <span key={idx} style={{ color: "#e6fff0" }}>{p}</span>;
      })}
    </>
  );
}
