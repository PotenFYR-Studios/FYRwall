// Command palette (SPEC 5.12): Ctrl/Cmd+K, Esc closes, arrows navigate,
// Enter opens. Searches all docs pages plus their h2/h3 sections.
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { DOCS, docBody } from "../content";

interface Hit {
  page: string;
  title: string;
  section?: string;
  anchor?: string;
  glyph: string;
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function buildIndex(): Hit[] {
  const hits: Hit[] = [];
  for (const d of DOCS) {
    hits.push({ page: d.slug, title: d.title, glyph: ">" });
    const body = docBody(d.slug) ?? "";
    for (const line of body.split("\n")) {
      if (line.startsWith("## ")) {
        hits.push({
          page: d.slug,
          title: d.title,
          section: line.slice(3).trim(),
          anchor: slugify(line.slice(3)),
          glyph: "S",
        });
      }
    }
  }
  return hits;
}

export function Palette() {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const index = useMemo(buildIndex, []);

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return index.slice(0, 14);
    return index
      .filter(
        (h) =>
          h.title.toLowerCase().includes(q) ||
          (h.section ?? "").toLowerCase().includes(q)
      )
      .slice(0, 14);
  }, [index, query]);

  useEffect(() => {
    const onOpen = () => setOpen(true);
    window.addEventListener("fyrwall:open-palette", onOpen);
    return () => window.removeEventListener("fyrwall:open-palette", onOpen);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen(false);
        return;
      }
      if (e.key === "Escape") {
        setOpen(false);
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelected((s) => Math.min(s + 1, results.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelected((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const hit = results[selected];
        if (hit) {
          setOpen(false);
          navigate(
            hit.anchor
              ? `/docs/${hit.page}#${hit.anchor}`
              : `/docs/${hit.page}`
          );
          setQuery("");
          setSelected(0);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    // autofocus each open
    requestAnimationFrame(() => inputRef.current?.focus());
    return () => window.removeEventListener("keydown", onKey);
  }, [open, results, selected, navigate]);

  if (!open) return null;

  return (
    <div
      className="palette-overlay"
      onClick={() => setOpen(false)}
      role="dialog"
      aria-modal="true"
      aria-label="Search documentation"
    >
      <div className="palette-panel" onClick={(e) => e.stopPropagation()}>
        <input
          ref={inputRef}
          className="palette-input"
          placeholder="Search docs..."
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setSelected(0);
          }}
          aria-label="Search query"
        />
        <div className="palette-results">
          {results.length === 0 && (
            <div className="px-3 py-6 text-center text-sm text-faint">
              No results.
            </div>
          )}
          {results.map((h, i) => (
            <button
              type="button"
              key={`${h.page}-${h.anchor ?? "page"}-${i}`}
              className={`palette-row${i === selected ? " selected" : ""}`}
              onMouseEnter={() => setSelected(i)}
              onClick={() => {
                setOpen(false);
                navigate(
                  h.anchor ? `/docs/${h.page}#${h.anchor}` : `/docs/${h.page}`
                );
                setQuery("");
                setSelected(0);
              }}
            >
              <span className="glyph">{h.glyph}</span>
              <span className="text-white">{h.title}</span>
              {h.section && <span className="text-muted">{h.section}</span>}
            </button>
          ))}
        </div>
        <div className="palette-foot">
          <span>arrows navigate</span>
          <span>Enter open</span>
          <span>Esc close</span>
        </div>
      </div>
    </div>
  );
}

export { slugify };
