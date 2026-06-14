# Qratum landing page — "Light Ledger"

The public landing page for **Qratum**. Direction: **Light Ledger** — a
neobrutalist light surface with the full vault-style animation set
(framer-motion typewriter hero, `sha256:` digest marquee, `qrt vault doctor`
terminal reveal, count-up stats).

This directory is a **self-contained Vite app that deploys independently** of the
Qratum Go runtime. Nothing here is part of the `qrt` binary, Go modules, or the
qratum runtime supply chain.

## Stack

- Vite 5 + React 18 + TypeScript
- framer-motion `12.40.0` (pinned)
- Tailwind CSS 3.4
- pnpm 10.11.0 · Node 20

## Local

```sh
pnpm install
pnpm dev        # http://127.0.0.1:7218
pnpm typecheck
pnpm build      # outputs dist/
```

## Files

- `src/drafts/ledger-light/` — the page and its widgets (`Page.tsx`,
  `Terminal.tsx`, `Typewriter.tsx`, `CountUp.tsx`, `icons.tsx`, `style.css`).
- `src/content.ts` — all product copy (single source of truth).
- `src/lib/motion.ts` — shared framer-motion variants.
- `vercel.json` — Vercel build config for this app.
- `BRIEF.md`, `BRIEF-FUSION.md` — design rationale (the rounds of directions
  that led to Light Ledger; historical).

`_discarded/` holds rejected draft directions from the design rounds. It is
gitignored and not built.

## Deploy (Vercel)

Delivery is **Vercel's git integration** — deploy is intentionally NOT done from
GitHub Actions (no Vercel token or third-party deploy action in CI is the
lower-risk choice for this repo).

One-time setup:

1. Import the repo into Vercel.
2. Set **Root Directory** = `design/landing-drafts` (so Vercel builds only this
   app, not the Go repo).
3. Framework Preset: **Vite** (auto-detected from `vercel.json`).
   - Install Command: `pnpm install --frozen-lockfile`
   - Build Command: `pnpm build`
   - Output Directory: `dist`
4. **Production Branch** = `main`.
5. Node version: Vercel reads `engines.node` (`20.x`) from `package.json`.

After that, every push to `main` ships to production and every PR gets a preview
URL — automatically.

## CI gate

`.github/workflows/landing.yml` runs `pnpm typecheck` + `pnpm build` on any
change under `design/landing-drafts/**`. It is scoped (does not run for Go-only
changes) and isolated from the runtime CI in `.github/workflows/ci.yml`. GitHub
Actions are pinned by commit SHA and `persist-credentials: false`, per the
repo supply-chain policy (`docs/supply-chain.md`).

## Supply-chain isolation

`make supply-chain` (the runtime policy check) scans only `.github/`,
`scripts/`, `Makefile`, and `go.*` — this directory is out of scope and has its
own `.gitignore` (`node_modules`, `dist`, `_discarded`). No npm/pip/curl
installers enter the qratum runtime pipeline from here.
