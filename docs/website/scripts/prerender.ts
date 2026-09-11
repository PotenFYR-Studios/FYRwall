// Custom static generation: renders the home route to static HTML so
// crawlers and no-JS visitors get the real content in the initial response.
// The site is hash-routed, so "/" is the only server-visible URL to render.
// Runs after `vite build`; no extra dependencies (vite + react-dom only).
import { createServer } from "vite";
import { renderToString } from "react-dom/server";
import { readFile, writeFile } from "node:fs/promises";
import React from "react";
import { MemoryRouter } from "react-router-dom";

const vite = await createServer({
  server: { middlewareMode: true },
  appType: "custom",
  logLevel: "error",
});

// react-router's NavLink warns about useLayoutEffect during SSR; harmless
// here (static snapshot, client re-renders fresh), so keep CI output clean.
const origError = console.error;
console.error = (...args: unknown[]) => {
  if (String(args[0]).includes("useLayoutEffect")) return;
  origError(...args);
};

try {
  const { default: App } = await vite.ssrLoadModule("/src/App.tsx");
  const html = renderToString(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(MemoryRouter, { initialEntries: ["/"] }, React.createElement(App))
    )
  );
  await vite.close();

  // import.meta.url is percent-encoded; decode it (path contains a space).
  const dist = decodeURIComponent(new URL("../dist/index.html", import.meta.url).pathname);
  const file = await readFile(dist, "utf8");
  if (!file.includes('<div id="root"></div>')) {
    throw new Error("root div placeholder not found in dist/index.html");
  }
  await writeFile(dist, file.replace('<div id="root"></div>', `<div id="root">${html}</div>`));
  console.log(`[prerender] home route rendered to dist/index.html (${html.length} bytes)`);
} catch (err) {
  await vite.close();
  throw err;
}
