import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Base path supports the hybrid GitHub Pages model: works at
// https://potenfyr-studios.github.io/FYRwall/ and at any custom domain
// served from the same artifact.
export default defineConfig({
  plugins: [react()],
  base: process.env.VITE_BASE ?? "/FYRwall/",
  build: { outDir: process.env.VITE_OUT_DIR ?? "dist" },
});
