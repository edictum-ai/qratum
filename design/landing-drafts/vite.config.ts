import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { qratum as q } from "./src/content";

// Design draft gallery — local dev only. Not wired into qratum runtime/CI.
//
// The public/llms.txt and llms-full.txt files are NOT committed: they are
// generated from src/content.ts at build time so the LLM/doc surfaces can never
// drift from the canonical messaging. Vite's config loader transpiles this TS
// (and its content.ts import), so we can build them straight from the data.

const here = fileURLToPath(new URL(".", import.meta.url));

const publishedOneLiner = `${q.oneLiner} Single Go binary (\`qrt\`), v${q.version}, install with \`brew install qratum\`. No cloud, no accounts, no telemetry.`;

function renderLlms(): string {
  return [
    "# Qratum",
    "",
    `> ${publishedOneLiner} Live site: ${q.siteUrl}`,
    "",
    q.whatItIs,
    "",
    "## Why",
    q.problem.body,
    "",
    "## Pillars",
    ...q.pillars.map((p) => `- ${p.name}: ${p.summary}`),
    "",
    "## Trust",
    `${q.trust.headline} ${q.trust.items.map((i) => i.body).join(" ")}`,
    "",
    "## Status",
    q.status.llms,
    "",
    "## Links",
    `- Website: ${q.siteUrl}`,
    `- Full version: ${q.siteUrl}/llms-full.txt`,
    `- Repository: ${q.repoUrl}`,
    `- Spec: ${q.repoUrl}/blob/main/SPEC.md`,
    "",
  ].join("\n");
}

function renderLlmsFull(): string {
  return [
    "# Qratum — full",
    "",
    `> ${publishedOneLiner}`,
    "",
    `Website: ${q.siteUrl}`,
    `Repository: ${q.repoUrl}`,
    "License: MIT",
    "",
    "## What it is",
    q.whatItIs,
    "",
    "## Why",
    q.problem.body,
    "",
    "## Pillars",
    ...q.pillars.flatMap((p) => [`### ${p.name}`, p.summary, ...p.points.map((pt) => `- ${pt}`), ""]),
    "## How it flows",
    ...q.loop.map((s) => `- ${s.step}: ${s.detail}`),
    "",
    "## Trust",
    q.trust.headline,
    ...q.trust.items.map((i) => `- ${i.title}: ${i.body}`),
    "",
    "## Status (honest)",
    q.status.llms,
    "",
    "## Roadmap",
    ...q.roadmap.map((r) => `- ${r.phase} — ${r.title} (${r.status}): ${r.detail}`),
    "",
    "## Honest boundaries",
    ...q.honest.map((h) => `- ${h}`),
    "",
    "## Quick start",
    "```",
    ...q.install,
    "qrt daemon run-once",
    "qrt sessions list",
    "```",
    "",
    "## Ecosystem",
    ...q.ecosystem.map((e) => `- ${e.name}: ${e.role}`),
    "",
    "## Links",
    `- Website: ${q.siteUrl}`,
    `- Repository: ${q.repoUrl}`,
    `- Spec: ${q.repoUrl}/blob/main/SPEC.md`,
    "",
  ].join("\n");
}

function llmsPlugin(): Plugin {
  let outDir = "dist";
  return {
    name: "qratum-llms-generate",
    apply: "build",
    configResolved(cfg) {
      outDir = cfg.build.outDir;
    },
    closeBundle() {
      const dir = resolve(here, outDir);
      writeFileSync(resolve(dir, "llms.txt"), renderLlms());
      writeFileSync(resolve(dir, "llms-full.txt"), renderLlmsFull());
    },
  };
}

export default defineConfig({
  plugins: [react(), llmsPlugin()],
  server: { host: "127.0.0.1", port: 7218 },
});
