import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Design draft gallery — local dev only. Not wired into qratum runtime/CI.
export default defineConfig({
  plugins: [react()],
  server: { host: "127.0.0.1", port: 7218 },
});
