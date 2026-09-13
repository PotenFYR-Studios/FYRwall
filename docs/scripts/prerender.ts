// Multi-page emit (the AuthCore pattern): after `vite build`, render
// every route to a real HTML file so direct refreshes, crawlers and
// no-JS visitors get full content plus per-route title, description,
// canonical, OG tags and JSON-LD.
//
//   /                    -> dist/index.html
//   /docs                -> dist/docs/index.html
//   /docs/<slug>         -> dist/docs/<slug>/index.html   (13 pages)
//   /examples            -> dist/examples/index.html
//   /about               -> dist/about/index.html
//
// Canonical URLs always carry the trailing slash: GitHub Pages 301s the
// bare path to the directory form, so that is the canonical target.
import { createServer } from "vite";
import { renderToString } from "react-dom/server";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import React from "react";
import { MemoryRouter } from "react-router-dom";

// DOCS/SITE_URL/routeMeta come from src/content.ts, which uses Vite's
// import.meta.glob - that only exists when Vite transforms the module, so
// it must be loaded through ssrLoadModule below, never imported directly
// (bun executing this script has no import.meta.glob).
let DOCS: Array<{ slug: string; title: string; description: string }> = [];
let SITE_URL = "https://fyrwall.docs.potenfyr.in";
let routeMeta: (path: string) => { title: string; description: string };

const here = dirname(fileURLToPath(import.meta.url));

// react-router's NavLink warns about useLayoutEffect during SSR; it is
// harmless here (static snapshot, client re-renders fresh).
const origError = console.error;
console.error = (...args: unknown[]) => {
  if (String(args[0]).includes("useLayoutEffect")) return;
  origError(...args);
};

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;");
}

function jsonLdScript(data: unknown): string {
  return `<script type="application/ld+json">\n${JSON.stringify(data)}\n</script>`;
}

function breadcrumb(path: string, ...crumbs: Array<{ name: string; item?: string }>) {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: crumbs.map((c, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: c.name,
      ...(c.item ? { item: c.item } : {}),
    })),
  };
}

function pageJsonLd(route: string): string {
  const clean = route.replace(/\/+$/, "") || "/";
  if (clean === "/") {
    // FAQPage lives in the shell; SoftwareApplication is site-wide.
    return "";
  }
  if (clean === "/docs") {
    return jsonLdScript({
      "@context": "https://schema.org",
      "@type": "CollectionPage",
      name: "FYRwall Documentation",
      url: `${SITE_URL}/docs/`,
      description: routeMeta("/docs").description,
      author: { "@type": "Organization", name: "PotenFYR Studios" },
    });
  }
  if (clean === "/examples" || clean === "/about") {
    return jsonLdScript({
      "@context": "https://schema.org",
      "@type": "WebPage",
      name: routeMeta(clean).title,
      url: `${SITE_URL}${clean}/`,
      description: routeMeta(clean).description,
      author: { "@type": "Organization", name: "PotenFYR Studios" },
    });
  }
  const slug = clean.replace(/^\/docs\//, "");
  const meta = DOCS.find((d) => d.slug === slug);
  if (!meta) return "";
  return [
    jsonLdScript({
      "@context": "https://schema.org",
      "@type": "TechArticle",
      headline: `${meta.title} | FYRwall Documentation`,
      description: meta.description,
      url: `${SITE_URL}/docs/${slug}/`,
      author: { "@type": "Organization", name: "PotenFYR Studios" },
      publisher: { "@type": "Organization", name: "PotenFYR Studios" },
    }),
    jsonLdScript(
      breadcrumb(`/docs/${slug}`,
        { name: "FYRwall", item: `${SITE_URL}/` },
        { name: "Docs", item: `${SITE_URL}/docs/` },
        { name: meta.title }
      )
    ),
  ].join("\n");
}

function outPathFor(route: string): string {
  const clean = route.replace(/\/+$/, "");
  // "" is the root route ("/" strips to ""); "docs" -> docs/index.html
  if (clean === "" || clean === "/") return "index.html";
  return `${clean.replace(/^\//, "")}/index.html`;
}

const vite = await createServer({
  server: { middlewareMode: true },
  appType: "custom",
  logLevel: "error",
});

try {
  const content = await vite.ssrLoadModule("/src/content");
  DOCS = content.DOCS;
  SITE_URL = content.SITE_URL;
  routeMeta = content.routeMeta;

  const ROUTES: string[] = [
    "/",
    "/docs",
    ...DOCS.map((d) => `/docs/${d.slug}`),
    "/examples",
    "/about",
  ];

  const { AppShell } = await vite.ssrLoadModule("/src/App.tsx");
  const base = (process.env.VITE_BASE ?? "/").replace(/\/+$/, "");

const dist = resolve(here, "..", process.env.VITE_OUT_DIR ?? "dist");
  console.log(`[prerender] dist: ${dist}`);
  const shellPath = resolve(dist, "index.html");
  const shell = await readFile(shellPath, "utf8");
  if (!shell.includes('<div id="root"></div>')) {
    throw new Error("root div placeholder not found in dist/index.html");
  }

  for (const route of ROUTES) {
    const body = renderToString(
      React.createElement(
        React.StrictMode,
        null,
        React.createElement(
          MemoryRouter,
          { initialEntries: [route], basename: base || undefined },
          React.createElement(AppShell)
        )
      )
    );
    const meta = routeMeta(route);

    const html = shell
      .replace(/<title>.*?<\/title>/, () => `<title>${esc(meta.title)}</title>`)
      .replace(
        /<meta name="description" content="[^"]*"/,
        () => `<meta name="description" content="${esc(meta.description)}"`
      )
      .replace(
        /<link rel="canonical" href="[^"]*"/,
        () => `<link rel="canonical" href="${SITE_URL}${route === "/" ? "/" : route.replace(/\/+$/, "") + "/"}"`
      )
      .replace(
        /<meta property="og:title" content="[^"]*"/,
        () => `<meta property="og:title" content="${esc(meta.title)}"`
      )
      .replace(
        /<meta property="og:description" content="[^"]*"/,
        () => `<meta property="og:description" content="${esc(meta.description)}"`
      )
      .replace(
        /<meta property="og:url" content="[^"]*"/,
        () => `<meta property="og:url" content="${SITE_URL}${route === "/" ? "/" : route.replace(/\/+$/, "") + "/"}"`
      )
      .replace(
        /<meta name="twitter:title" content="[^"]*"/,
        () => `<meta name="twitter:title" content="${esc(meta.title)}"`
      )
      .replace(
        /<meta name="twitter:description" content="[^"]*"/,
        () => `<meta name="twitter:description" content="${esc(meta.description)}"`
      )
      .replace("</head>", `${pageJsonLd(route)}\n</head>`)
      .replace('<div id="root"></div>', () => `<div id="root">${body}</div>`);

    const rel = outPathFor(route);
    const file = resolve(dist, rel);
    await mkdir(dirname(file), { recursive: true });
    await writeFile(file, html);
    console.log(`[prerender] ${route} -> dist/${rel} (${body.length} bytes)`);
  }

  await vite.close();
} catch (err) {
  await vite.close();
  throw err;
}
