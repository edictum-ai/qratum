import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";

/**
 * Dark Ledger — faux terminal running `qrt vault doctor`, adapted from the
 * vault draft. Lines reveal one-by-one on mount (once). Reduced-motion shows
 * the full output instantly with no reveal delay. The blinking caret is gated
 * behind reduced-motion.
 *
 * Illustrative terminal chrome for the documented `qrt vault doctor` /
 * `backup --verify` motif — not a claim beyond content.ts.
 */

type LineKind = "prompt" | "out" | "ok" | "warn" | "muted" | "head";

type Line = { kind: LineKind; text: string };

const SCRIPT: Line[] = [
  { kind: "prompt", text: "qrt vault doctor" },
  { kind: "head", text: "Qratum Vault · integrity check" },
  { kind: "muted", text: "────────────────────────────────────────────" },
  { kind: "out", text: "blob store      ~/.qratum/vault/blobs" },
  { kind: "ok", text: "objects         1,284   (deduped by sha256)" },
  { kind: "ok", text: "hooks           SessionEnd capture installed" },
  { kind: "warn", text: "backfill        3 transcripts unarchived → run `qrt vault backfill`" },
  { kind: "muted", text: "────────────────────────────────────────────" },
  { kind: "prompt", text: "qrt vault backup --verify" },
  { kind: "head", text: "Verifying backup provenance" },
  { kind: "ok", text: "sha256:9f3c…e71a  ✓ verified   adp-export/ref" },
  { kind: "ok", text: "sha256:4b18…0c2d  ✓ verified   raw/session-7221" },
  { kind: "ok", text: "sha256:de07…aa9f  ✓ verified   redacted/rev-3" },
  { kind: "muted", text: "────────────────────────────────────────────" },
  { kind: "ok", text: "vault: OK   0 unverified · 0 raw routes exposed" },
];

const COLOR: Record<LineKind, string> = {
  prompt: "#f5d90a",
  out: "#e6fff0",
  ok: "#f5d90a",
  warn: "#f5b73d",
  muted: "#8a9aa0",
  head: "#34d1ff",
};

function prefersReduced() {
  return (
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}

export default function Terminal() {
  const [visible, setVisible] = useState<number>(() =>
    prefersReduced() ? SCRIPT.length : 0
  );
  const [typingDone, setTypingDone] = useState<boolean>(() => prefersReduced());
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (prefersReduced()) {
      setVisible(SCRIPT.length);
      setTypingDone(true);
      return;
    }
    let i = 0;
    let timer: ReturnType<typeof setTimeout>;
    // Librarian-calm rhythm — longer pauses than the vault sibling.
    const tick = () => {
      i += 1;
      setVisible(i);
      if (i >= SCRIPT.length) {
        setTypingDone(true);
        return;
      }
      const last = SCRIPT[i - 1];
      const delay = last.kind === "prompt" || last.kind === "head" ? 460 : 210;
      timer = setTimeout(tick, delay);
    };
    timer = setTimeout(tick, 650);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="dl-terminal" ref={ref} aria-label="Simulated qrt vault doctor terminal output">
      <div className="dl-terminal-bar">
        <span className="dl-term-dot" style={{ background: "#d6453d" }} />
        <span className="dl-term-dot" style={{ background: "#f5d90a" }} />
        <span className="dl-term-dot" style={{ background: "#3fdd8a" }} />
        <span className="dl-mono dl-terminal-label">qrt — vault doctor</span>
        <span style={{ flex: 1 }} />
        <span className="dl-mono dl-terminal-host">127.0.0.1</span>
      </div>

      <div className="dl-terminal-body dl-mono">
        {SCRIPT.slice(0, visible).map((line, idx) => {
          const isPrompt = line.kind === "prompt";
          const isHead = line.kind === "head";
          return (
            <motion.div
              key={idx}
              initial={prefersReduced() ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.18, ease: [0.22, 1, 0.36, 1] }}
              style={{ color: COLOR[line.kind], display: "flex", gap: 8 }}
            >
              {isPrompt && <span style={{ color: "#f5d90a" }}>$</span>}
              {isHead && <span style={{ color: "#34d1ff" }}>»</span>}
              {!isPrompt && !isHead && <span style={{ opacity: 0 }}>·</span>}
              <span>{line.text}</span>
            </motion.div>
          );
        })}

        {typingDone ? (
          <div style={{ display: "flex", gap: 8, color: "#f5d90a" }}>
            <span>$</span>
            <span className="dl-caret" aria-hidden="true" />
          </div>
        ) : (
          <div style={{ display: "flex", gap: 8 }}>
            <span style={{ opacity: 0 }}>·</span>
            <span className="dl-caret" aria-hidden="true" />
          </div>
        )}
      </div>
    </div>
  );
}
