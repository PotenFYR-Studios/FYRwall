import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev proxy targets the local fyrwall server; production builds are
// embedded into the Go binary (no Node on the host, spec section 3.2).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:7443",
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
});
