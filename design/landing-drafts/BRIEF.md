# Qratum Landing Drafts — Design Brief

You are building **one** of five landing-page drafts for **Qratum**.
All five render the **same product, the same words, the same section order** —
only the design language differs. That is the whole point: Arnold will compare
the five directions and pick one to become the visual language for the public
landing page *and* the future local app shell.

## What Qratum is (read this even if you think you know)

Qratum is a **local-first library, vault, and review pipeline for AI coding
sessions**. It captures, preserves, normalizes, redacts, reviews, and searches
every AI session — without ever uploading raw transcripts.

- **The problem:** AI coding tools delete sessions. Claude Code purges
  transcripts after ~30 days. Months of work vanish, and nothing records
  *provenance* (where the data came from).
- **The wedge:** the **Vault** — content-addressed preservation so a transcript
  deleted tomorrow is recoverable. Passive value that survives abandonment.
- **The refinery:** on-demand normalize → deterministic redaction → evidence →
  review → report → corpus export. Runs only when asked.
- **The character:** local-first, single Go binary (`qrt`), no accounts, no
  cloud, no telemetry, deterministic, content-addressed, trust-boundaried,
  consent-gated. This is a security-adjacent, developer, archival tool.

Do **not** invent product claims. Pull all copy from `src/content.ts`.

## Files you own (write ONLY these)

```
src/drafts/<your-slug>/Page.tsx        ← default export: your whole landing page
src/drafts/<your-slug>/style.css       ← optional, your scoped CSS (import it in Page.tsx)
```

- `<your-slug>` is one of: `vault`, `librarian`, `refinery`, `ledger`, `quiet`.
- Wrap your entire page in a single root element with the class
  `.<your-slug>-root` and scope any custom CSS to it. Do **not** use raw global
  selectors that could leak into other drafts.
- You may split sub-components into more files inside your own folder.

## Files you must NOT modify

Anything outside `src/drafts/<your-slug>/`. In particular:
`src/App.tsx`, `src/main.tsx`, `src/content.ts`, `src/drafts/registry.ts`,
`src/lib/motion.ts`, `src/index.css`, config files, `package.json`. The router
already lazy-imports your `Page.tsx` default export at `/d/<your-slug>`.

## Shared imports you should use

```tsx
import { qratum } from "../../content";              // all copy lives here
import { fadeUp, fadeIn, stagger, viewport, easeOut } from "../../lib/motion";
import { motion } from "framer-motion";              // pinned at 12.40.0
```

Tailwind is configured globally (v3.4) with font families: `font-sans` (Inter),
`font-mono` (JetBrains Mono), `font-serif` (Fraunces), `font-display`
(Space Grotesk), `font-reading` (Lora). These fonts are already loaded in
`index.html` — **do not add `<link>` tags**; if you need another weight it is
already covered, otherwise adapt.

## Required sections (ALL of these, in this order, for fair comparison)

Use `qratum.*` fields as the source for each. Render real content — no lorem.

1. **Top nav** — wordmark `qratum`, 2–3 links (e.g. Vault / Roadmap / Spec),
   one CTA button (`qratum.cta`).
2. **Hero** — `qratum.tagline` + `qratum.oneLiner`, a problem hook
   (`qratum.problem`), and the two CTAs. Plus your direction's signature
   visual motif (see your spec).
3. **Problem** — expand `qratum.problem` (ephemeral sessions, 30-day purge).
4. **Three pillars** — map over `qratum.pillars` (Vault / Refinery / Provenance),
   each with its `summary` + `points`.
5. **How it works** — render `qratum.loop` (Capture → Archive → Review →
   Library → Corpus) as a flow/steps visual.
6. **Trust / Security** — `qratum.trust.headline` + the four `qratum.trust.items`.
7. **Stats band** — render `qratum.stats` (4 stats).
8. **Roadmap** — render `qratum.roadmap` (Milestone A done → P0 now → P1 next →
   P2–P5 later). Show status honestly (done/now/next/later). This is a
   pre-1.0, spec-phase product — do **not** imply features are shipped that are
   still on the roadmap.
9. **Install / CTA** — `qratum.install` (code block) + `qratum.cta`.
10. **Footer** — `qratum.honest` (the three honesty lines), the
    `qratum.identity` chips, and `qratum.ecosystem` roles. Keep Ductum as
    internal (it is the internal agent factory; do not headline it).

## Quality bar (non-negotiable)

- **Distinctive.** Avoid generic "AI SaaS" aesthetics (avoid the default
  purple→pink gradient hero, avoid stock 3D, avoid emoji as icons). Your
  direction must look intentional and authored. Use inline SVG line icons
  (lucide-style), not emoji.
- **Motion with `framer-motion` (12.40.0).** Real scroll reveals
  (`whileInView` + `viewport`), staggered children, purposeful hovers. Not
  decorative spam.
- **Accessibility.** Real semantic HTML (`<nav> <main> <section> <footer>`),
  heading order, `alt`/`aria-label` on icon-only buttons, focus-visible styles,
  color contrast ≥ 4.5:1 for body text. Buttons/links get `cursor-pointer`.
- **Reduced motion.** Honor `prefers-reduced-motion` for anything non-essential
  (a global rule already exists in `index.css`; additionally gate your big
  hero animations behind it).
- **Responsive.** Look correct at 375 / 768 / 1024 / 1440 px. Test mentally at
  mobile; collapse grids to single column.
- **Reusable as an app shell.** This is also a probe for the local app's visual
  language — keep a coherent nav + content rhythm that could become an app chrome.
- **No external runtime deps.** Use only what's in `package.json`. Do not add
  npm packages. Inline your SVGs and CSS. No CDN scripts beyond the fonts
  already in `index.html`.

## The five directions

Each agent owns exactly one. (Shown together so you avoid stepping on a sibling.)

### 1. `vault` — "The Vault" · Terminal / HUD · DARK
- **Pattern:** Hero-Centric + Interactive product demo (a terminal as hero).
- **Style:** Dark Mode (OLED) + HUD/Sci-Fi FUI; restrained cyberpunk minimalism.
- **Colors:** canvas `#05070a`; surface `#0b0f14`; text `#e6fff0` / muted
  `#b8c7be`; accent phosphor green `#3fdd8a`; secondary cyan `#34d1ff`;
  amber `#f5b73d`; hairlines `rgba(63,221,138,0.18)`.
- **Type:** JetBrains Mono for wordmark/labels/headlines (mono-forward at
  display size), Inter for body. Uppercase tracked labels.
- **Effects:** a faux terminal window as hero centerpiece running
  `qrt vault doctor` output (line-by-line); blinking caret (reduced-motion
  aware); faint scanline overlay; glass panels with 1px neon hairline + soft
  inner glow; faint grid bg; `sha256:…` digest chips; ASCII box-drawing borders.
- **Motion:** typewriter hero subtitle, staggered reveals, count-up stats,
  hover glow, spring.
- **Anti-patterns:** NO purple/pink AI gradients; one accent (green) + restrained
  cyan only; keep AA contrast; stay minimal — terminal austerity, not clutter.
- **Motif:** `qrt vault doctor` terminal output; content-addressed digest chips.

### 2. `librarian` — "The Librarian" · Editorial / Swiss · LIGHT
- **Pattern:** Storytelling-Driven + Editorial Grid / Magazine.
- **Style:** Swiss Modernism 2.0 + Editorial Grid + restrained minimalism.
- **Colors:** paper `#f7f4ee` / `#fbf9f4`; ink `#1a1a1a`; muted `#6b6258`;
  accent oxblood `#7c2d2d` (or deep emerald `#1f5f4a` — pick one); rules
  `rgba(0,0,0,0.12)`.
- **Type:** Fraunces (opsz serif) for display H1/H2 + pull quotes; Inter for
  body/UI; JetBrains Mono only for tiny catalog metadata. Large display sizes;
  body measure ~62ch; strong hierarchy.
- **Effects:** horizontal/vertical rule lines dividing sections; numbered
  sections `01 / 02 / 03` like a catalog; ample whitespace; a duotone archival
  motif (CSS gradient or inline SVG). Calm. No gradients, no shadows.
- **Motion:** slow + quiet. Long fade + slight translate (y:24→0, ~0.8s).
  No bouncy springs. Respect reduced motion.
- **Anti-patterns:** NO drop shadows / glass / neon / emoji; whitespace IS the
  design; one accent only; no busy.
- **Motif:** a library catalog card / index; provenance as a citation.

### 3. `refinery` — "The Refinery" · Bento / modern SaaS · LIGHT
- **Pattern:** Feature-Rich Showcase (bento) + Conversion-Optimized.
- **Style:** Bento Box Grid + Dimensional Layering + soft accents.
- **Colors:** canvas `#f7f8fb`; surface `#ffffff`; ink `#0b1020`; muted
  `#5b6478`; accent indigo `#4f46e5`; secondary `#06b6d4`; soft layered shadows
  `rgba(11,16,32,0.06–0.12)`.
- **Type:** Space Grotesk for display; Inter for body; JetBrains Mono for
  code/labels. Tight, modern.
- **Effects:** bento grid of feature cards (rounded-2xl, soft layered shadows,
  subtle gradient fills); hover lift + slight tilt; inline line icons;
  micro-interactions on loop steps; **an animated pipeline diagram** (nodes
  connected: Capture→Archive→Review→Library→Corpus) for "how it works";
  count-up stats.
- **Motion:** staggered bento entrance; springy hovers (stiffness ~300);
  draw-in connectors; count-ups. Modern, snappy.
- **Anti-patterns:** avoid generic stock-SaaS purple-gradient cliché (keep
  indigo restrained, lots of white); no 3D; crisp not busy.
- **Motif:** an animated pipeline/flow diagram.

### 4. `ledger` — "The Ledger" · Neubrutalism · LIGHT
- **Pattern:** Minimal & Direct + bold feature blocks.
- **Style:** Neubrutalism (restrained, professional).
- **Colors:** paper `#f4f1e8`; ink `#0b0b0b`; bold blocks: yellow `#f5d90a`,
  blue `#2563eb`, clay red `#d6453d`; **hard flat offset shadows**
  (`box-shadow: 6px 6px 0 #000`, no blur).
- **Type:** Space Grotesk / heavy weight for huge bold headlines (tight
  tracking, oversized); Inter body; JetBrains Mono for manifest rows. ALL-CAPS
  tracked labels.
- **Effects:** 2–3px solid black borders on everything; hard offset shadows;
  **data-class badges as sticker chips** (`raw` `redacted` `review` …); a
  **manifest/receipt table** (monospace rows: `kind | digest | size | status`);
  a marquee ticker of digests; exposed grid lines.
- **Motion:** snappy + punchy. Spring with low damping. Marquee. Hard, abrupt
  transitions — no soft fades.
- **Anti-patterns:** NO gradients / glass / soft shadows (hard offset shadows
  only); max ~3–4 block colors; brutal but legible.
- **Motif:** a manifest/receipt table of content-addressed items; sticker
  data-class badges.

### 5. `quiet` — "Quiet Trust" · Soft / premium · LIGHT
- **Pattern:** Trust & Authority + Minimal & Direct.
- **Style:** Soft UI Evolution + Minimalism (warm, premium privacy tool).
- **Colors:** canvas `#f3efe8` / `#faf7f1`; surface `#fffdf9`; ink `#2b2b2b`;
  muted `#8a8278`; accent sage/olive `#7c8a5a`; secondary clay `#b08968`; soft
  diffused shadows `rgba(120,110,90,0.10–0.16)`; very gentle warm gradients.
- **Type:** Lora (serif) for display/headings; Inter for body/UI. Generous
  line-height; comfortable measure.
- **Effects:** soft rounded cards (rounded-3xl) with diffused shadows; very
  gentle gradients; lots of negative space; subtle hairline dividers; small
  organic inline-SVG accents. Calm hover scale (1.01–1.02).
- **Motion:** gentle. Long fades (~0.7s), soft eases, slow hover scale.
  Strongly respect reduced motion. Nothing flashy.
- **Anti-patterns:** NO hard borders/shadows; no neon; no brutalism; warm not
  cold-blue; no clutter.
- **Motif:** a soft "vault/safe" or calm shield; emphasize "stays local" and
  "survives abandonment gracefully."

## How to run (already wired)

```sh
cd design/landing-drafts
pnpm install      # already done
pnpm dev          # http://127.0.0.1:7218  → gallery lists all five
```

Your draft is at `/d/<your-slug>`. The gallery at `/` links to it.

## Definition of done for your draft

- `Page.tsx` is a complete, self-contained landing page (all 10 sections).
- `pnpm build` succeeds with your file in place.
- Looks polished and responsive at mobile + desktop.
- Motion is real and reduced-motion-safe.
- No content invented beyond `qratum.*`; roadmap shown honestly.
- Scoped to `.<your-slug>-root`; nothing touched outside your folder.
```
