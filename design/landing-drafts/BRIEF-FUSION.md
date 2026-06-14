# Qratum Landing Drafts — Round 2: Fusions

Arnold reviewed round 1 and gave this fusion brief:

> I like **the ledger** (look) + **features of the vault** (widgets) +
> **the movements of the letter** (motion → the *librarian*'s calm, refined motion,
> read literally as kinetic typography too) + **wording of refinery** (microcopy voice).

You are building **one** of four new drafts that **recombine those four liked things**
in different proportions. The goal is to find the synthesis that becomes Qratum's
visual language for the landing page *and* the future local app shell.

## The fusion recipe (every new draft draws from all four)

1. **VISUAL BASE = the ledger** (`src/drafts/ledger/Page.tsx` + `style.css`).
   Neobrutalism: 2–3px solid borders, hard flat offset shadows
   (`box-shadow: 6px 6px 0 #000`), bold display type, data-class **sticker chips**,
   a **manifest/receipt table** (mono rows: `kind | class | digest | size | status`).

2. **MOTION = the librarian** (`src/drafts/librarian/Page.tsx` + `style.css` — study
   its `reveal`/`stagger` variants). **Calm and refined**: long fades (~0.8s), a slow
   ease (`[0.22,1,0.36,1]`), quiet staggered children. **No punchy springs** as the
   default. Reserve snappier motion only for the interactive widgets below and an
   optional kinetic-typography headline. Strongly honor `prefers-reduced-motion`.

3. **WIDGETS = the vault** (`src/drafts/vault/Page.tsx`, `Terminal.tsx`,
   `CountUp.tsx`, `Typewriter.tsx`, `icons.tsx`). **Copy these into YOUR folder and
   adapt them** to your aesthetic — do **not** import across draft folders (keep each
   draft self-contained/isolated). Each new draft MUST include:
   - a `qrt vault doctor`-style **terminal block** (line-by-line reveal, blinking caret
     gated behind reduced-motion),
   - **`sha256:` digest chips**,
   - **count-up stats**,
   - a **copy-to-clipboard install block**.

4. **MICROCOPY = the refinery** (`src/drafts/refinery/Page.tsx`). Adopt its crisp,
   technical voice for eyebrows / badges / labels — e.g. `single binary · local-first`,
   `refinery · on-demand`, `normalize → redact → evidence → review → corpus`,
   `eligibility-checked`, `consent: explicit`. Keep these as small labels, not body copy.

All copy still comes from `qratum.*` in `src/content.ts`. Same **10 required sections**
and **quality bar** as `BRIEF.md` (read it). Roadmap stays honest (pre-1.0; only
Milestone A done; P0 now). Ductum stays internal/background in the footer.

## READ BEFORE WRITING
1. `src/content.ts` — all copy.
2. `src/drafts/ledger/Page.tsx` + `ledger/style.css` — the bones.
3. `src/drafts/vault/Page.tsx` + `vault/Terminal.tsx`, `vault/CountUp.tsx`,
   `vault/Typewriter.tsx`, `vault/icons.tsx` — the widgets to adapt.
4. `src/drafts/librarian/Page.tsx` + `librarian/style.css` — the motion.
5. `src/drafts/refinery/Page.tsx` — the microcopy voice.
6. `BRIEF.md` — quality bar, hard rules, required sections.

## The four directions

### `quiet-brutal` — "Quiet Brutalism" (the canonical calm fusion) · LIGHT
- **Bones:** ledger neobrutalism on paper `#f4f1e8`, ink `#0b0b0b`, 2.5px solid black
  borders, hard offset shadows, Space Grotesk heavy display + ALL-CAPS tracked labels,
  sticker data-class chips, a manifest table.
- **Twist:** the motion is the librarian's — slow, quiet, deliberate long fades and
  refined stagger. This is the core experiment: **calm motion on a brutalist base**.
- **Accent:** clay red `#d6453d` (one bold accent + ink).
- Include all four vault widgets, framed in bordered brutalist panels. Refinery microcopy.

### `manifest` — "Manifest" (receipt / terminal, ops feel) · LIGHT
- **Bones:** monospace-**forward** (JetBrains Mono as a primary face), paper `#f5f3ec`,
  ink `#0b0b0b`, hard borders, a **perforated receipt edge** (dashed border motif).
- **Hero:** a `qrt vault archive` run that produces a **manifest/receipt** (mono rows
  `kind | class | digest | size | status`) — the manifest IS the hero artifact. Include
  the `qrt vault doctor` terminal widget too.
- **Accent:** receipt/printout teal-green `#1f6f5c`.
- Motion = librarian calm + vault terminal typewriter + count-ups. Refinery microcopy.
  Lean "system / ops / provenance" — Qratum as a content-addressed ledger of record.

### `editorial` — "Editorial Brutalism" (serif + grotesk, kinetic type) · LIGHT
- **Bones:** brutalist grid but with **editorial restraint** — more whitespace inside the
  hard borders, strong type hierarchy, numbered sections (`01 / 02 / 03`).
- **Typography:** **mix Fraunces serif headlines with bold Space Grotesk** — the literal
  "letter/movement" reading made visible.
- **Hero:** a **kinetic-typography headline** — words/letters ease in with the librarian's
  slow curve and a refined stagger (reduced-motion: fall back to a plain fade).
- **Accent:** emerald `#1f5f4a`. Paper `#f6f3ec`.
- Include the vault widgets as framed brutalist cards. Refinery microcopy.

### `dark-ledger` — "Dark Ledger" (neobrutalism on OLED) · DARK
- **Bones:** the SAME fusion, but on OLED dark `#06080c`. Hard borders in a light/
  accent stroke (e.g. `2px solid #f5d90a` or `#e6fff0`), hard offset shadows in the
  accent, Space Grotesk heavy display, sticker chips, manifest table in mono (light text
  on dark). This surface suits the vault widgets best.
- **Include ALL vault widgets** prominently: the `qrt vault doctor` terminal, digest chips,
  count-ups, copy-clipboard, phosphor accent glows kept tasteful (no neon overload).
- **Accent:** yellow `#f5d90a` on dark; secondary cyan `#34d1ff` restrained.
- Motion = librarian calm (restrained on dark). Refinery microcopy.

## HARD RULES (unchanged from BRIEF.md)
- Write **only** inside `src/drafts/<your-slug>/`. Touch nothing else.
- `framer-motion` 12.40.0, Tailwind, the already-loaded fonts. **No new packages, no
  CDN/`<link>` tags, inline all SVG.** Copy+adapt vault widgets into your own folder;
  never import from a sibling draft folder.
- Accessibility (semantic HTML, heading order, aria-labels, focus-visible, contrast
  ≥ 4.5:1, cursor-pointer), responsive 375–1440, reduced-motion safe.
- Honest roadmap; Ductum stays internal in the footer.

## VERIFY (required — the dev server is ALREADY running; do NOT start another)
From `/Users/acartagena/project/qratum/design/landing-drafts` run:
```
pnpm typecheck
```
(read-only `tsc --noEmit`; safe to run alongside others). Fix YOUR file until it passes.
**Do NOT run `pnpm dev` (a server is already up on :7218) or `pnpm build` (dist race).**

## REPORT BACK
(a) typecheck pass/fail, (b) 3–4 line summary of how you fused the four inputs +
key motion/widgets, (c) files touched (all under `src/drafts/<your-slug>/`). Say plainly
if blocked; never claim success if typecheck failed.
