// Docs reader: grouped sidebar + article (max-w-6xl, fix-round-2 W1) + right
// TOC with violet active bar, prev/next pagination, [ and ] page jumps.
import { useEffect, useMemo, useState } from "react";
import { Link, NavLink, useLocation, useParams } from "react-router-dom";
import { DOCS, DOC_GROUPS, docBody, docBySlug } from "../content";
import { renderMarkdown, slugify, type Heading } from "../components/md";
import NotFound from "./NotFound";

function SidebarNav({ active }: { active: string }) {
  return (
    <nav aria-label="Docs sections">
      {DOC_GROUPS.map((group) => (
        <details key={group} className="side-group" open>
          <summary className="side-title">{group}</summary>
          {DOCS.filter((d) => d.group === group).map((d) => (
            <NavLink
              key={d.slug}
              to={`/docs/${d.slug}`}
              className={({ isActive }) => `side-link${isActive ? " active" : ""}`}
            >
              {d.title}
            </NavLink>
          ))}
        </details>
      ))}
    </nav>
  );
}

function Toc({ headings }: { headings: Heading[] }) {
  const [activeId, setActiveId] = useState<string>("");
  const tocItems = useMemo(() => headings.filter((h) => h.level === 2 || h.level === 3), [headings]);

  useEffect(() => {
    if (tocItems.length === 0) return;
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
            break;
          }
        }
      },
      { rootMargin: "-80px 0px -65% 0px" }
    );
    for (const h of tocItems) {
      const el = document.getElementById(h.id);
      if (el) observer.observe(el);
    }
    return () => observer.disconnect();
  }, [tocItems]);

  if (tocItems.length < 3) return null;
  return (
    <nav className="toc" aria-label="On this page">
      <div className="toc-title grad-text">On this page</div>
      {tocItems.map((h) => (
        <a
          key={h.id}
          href={`#${h.id}`}
          className={`toc-item toc-h${h.level}${activeId === h.id ? " active" : ""}`}
          aria-current={activeId === h.id ? "true" : undefined}
        >
          {h.text}
        </a>
      ))}
    </nav>
  );
}

export default function DocPage() {
  const { slug } = useParams();
  const location = useLocation();
  const meta = docBySlug(slug);
  const body = slug ? docBody(slug) : undefined;

  const rendered = useMemo(
    () => (body === undefined ? null : renderMarkdown(body)),
    [body]
  );

  // Anchor scrolling for palette / TOC hits.
  useEffect(() => {
    if (!location.hash) return;
    const el = document.getElementById(location.hash.slice(1));
    if (el) el.scrollIntoView({ block: "start" });
  }, [location.hash, rendered]);

  // [ and ] jump to previous/next page.
  useEffect(() => {
    if (!meta) return;
    const idx = DOCS.findIndex((d) => d.slug === meta.slug);
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement | null;
      if (t && (t.tagName === "INPUT" || t.tagName === "TEXTAREA")) return;
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      if (e.key === "[") {
        const prev = DOCS[(idx - 1 + DOCS.length) % DOCS.length];
        window.location.assign(`/docs/${prev.slug}`);
      } else if (e.key === "]") {
        const next = DOCS[(idx + 1) % DOCS.length];
        window.location.assign(`/docs/${next.slug}`);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [meta]);

  if (!meta || rendered === null) return <NotFound />;

  const idx = DOCS.findIndex((d) => d.slug === meta.slug);
  const prev = idx > 0 ? DOCS[idx - 1] : null;
  const next = idx < DOCS.length - 1 ? DOCS[idx + 1] : null;

  return (
    <div className="layout">
      <aside className="layout-side">
        <SidebarNav active={meta.slug} />
      </aside>

      <article className="md-body min-w-0 max-w-6xl">
        <div className="crumbs">
          <Link to="/docs">Docs</Link> / {meta.group} / {meta.title}
        </div>
        {rendered.nodes}
        <nav className="pager" aria-label="Pagination">
          {prev ? (
            <Link to={`/docs/${prev.slug}`} className="prev">
              <span className="pager-label">Previous</span>
              <span className="pager-title">{prev.title}</span>
            </Link>
          ) : (
            <span className="flex-1" />
          )}
          {next ? (
            <Link to={`/docs/${next.slug}`} className="next">
              <span className="pager-label">Next</span>
              <span className="pager-title">{next.title}</span>
            </Link>
          ) : (
            <span className="flex-1" />
          )}
        </nav>
      </article>

      <aside className="layout-toc">
        <Toc headings={rendered.headings} />
      </aside>
    </div>
  );
}

export { SidebarNav, slugify };
