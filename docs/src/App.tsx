import { useEffect } from "react";
import { BrowserRouter, Route, Routes, useLocation } from "react-router-dom";
import { routeMeta } from "./content";
import { SiteFooter, SiteHeader } from "./components/SiteChrome";
import { Palette } from "./components/Palette";
import Home from "./pages/Home";
import DocsHub from "./pages/DocsHub";
import DocPage from "./pages/DocPage";
import Examples from "./pages/Examples";
import About from "./pages/About";
import NotFound from "./pages/NotFound";

// Base path is hybrid: "/" on the custom domain, "/FYRwall" on the
// potenfyr-studios.github.io fallback. Set by the Pages workflow.
const BASE = (import.meta.env.BASE_URL ?? "/").replace(/\/+$/, "");

function RouteEffects() {
  const location = useLocation();
  useEffect(() => {
    if (!location.hash) window.scrollTo(0, 0);
    // Keep tabs/history titles in sync on client-side navigation; the
    // prerenderer bakes the same titles into the static HTML.
    document.title = routeMeta(location.pathname).title;
  }, [location.pathname, location.hash]);
  return null;
}

/** Layout + routes only, no router - the prerenderer wraps it in a MemoryRouter. */
export function AppShell() {
  return (
    <div className="relative flex min-h-screen flex-col">
      <SiteHeader />
      <main className="relative z-[1] flex-1">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/docs" element={<DocsHub />} />
          <Route path="/docs/:slug" element={<DocPage />} />
          <Route path="/examples" element={<Examples />} />
          <Route path="/about" element={<About />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      <SiteFooter />
      <Palette />
      <RouteEffects />
    </div>
  );
}

export default function App() {
  return (
    <BrowserRouter basename={BASE || undefined}>
      <AppShell />
    </BrowserRouter>
  );
}
