// SSR render probe for the chosen landing page. Mounts ledger-light server-side
// via Vite's ssrLoadModule + renderToStaticMarkup. Run: node scripts/ssr-check.mjs
import { createServer } from "vite";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";

const vite = await createServer({
  server: { middlewareMode: true },
  appType: "custom",
  logLevel: "error",
});

try {
  const mod = await vite.ssrLoadModule("/src/drafts/ledger-light/Page.tsx");
  const Page = mod.default;
  if (typeof Page !== "function") throw new Error("no default export");
  const html = renderToStaticMarkup(React.createElement(Page));
  const checks = {
    wordmark: html.includes("qratum") || html.includes("QRATUM"),
    hasContent: html.length > 2000,
    noLorem: !/lorem ipsum/i.test(html),
  };
  const ok = checks.wordmark && checks.hasContent && checks.noLorem;
  console.log(
    `${ok ? "OK  " : "WARN"} ledger-light  ${String(html.length).padStart(6)} chars  ${JSON.stringify(checks)}`
  );
  await vite.close();
  process.exit(ok ? 0 : 1);
} catch (e) {
  console.error("FAIL ledger-light:", e && e.message ? e.message : e);
  await vite.close();
  process.exit(1);
}
