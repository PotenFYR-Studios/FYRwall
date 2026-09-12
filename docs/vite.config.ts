import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// Docs site for https://fyrwall.docs.potenfyr.in (GitHub Pages).
// Base path stays hybrid: the custom domain serves at "/", while the
// potenfyr-studios.github.io/FYRwall/ fallback serves under /FYRwall/.
// Every route is statically prerendered after the bundle step (see
// scripts/prerender.ts, the multi-page emit pattern from AuthCore) so
// direct refreshes return real HTML with per-route meta.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: process.env.VITE_BASE ?? "/",
  build: { outDir: "dist", emptyOutDir: true },
});
