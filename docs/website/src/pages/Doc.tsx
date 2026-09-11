import { useParams, Link, NavLink } from "react-router-dom";
import { useEffect, useState } from "react";

const BASE = import.meta.env.BASE_URL ?? "/";

// Lightweight markdown rendering: headings, code fences, inline code,
// bold/italic, links and lists. No external dependency, keeps the site
// fully offline-friendly like the product itself.
function renderMarkdown(md: string): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  const lines = md.split("\n");
  let i = 0;
  let key = 0;

  const inline = (text: string): React.ReactNode[] => {
    // inline code, bold, italic, links - order matters
    const parts: React.ReactNode[] = [];
    const re = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\*[^*]+\*)|(\[[^\]]+\]\([^)]+\))/g;
    let last = 0;
    let m: RegExpExecArray | null;
    let k = 0;
    while ((m = re.exec(text)) !== null) {
      if (m.index > last) parts.push(text.slice(last, m.index));
      const tok = m[0];
      if (tok.startsWith("`")) {
        parts.push(
          <code key={k++} className="bg-zinc-800 text-orange-300 rounded px-1.5 py-0.5">
            {tok.slice(1, -1)}
          </code>
        );
      } else if (tok.startsWith("**")) {
        parts.push(<strong key={k++}>{tok.slice(2, -2)}</strong>);
      } else if (tok.startsWith("*")) {
        parts.push(<em key={k++}>{tok.slice(1, -1)}</em>);
      } else {
        const lm = /\[([^\]]+)\]\(([^)]+)\)/.exec(tok)!;
        parts.push(
          <a key={k++} href={lm[2]} className="text-orange-400 hover:underline" target={lm[2].startsWith("http") ? "_blank" : undefined} rel="noopener noreferrer">
            {lm[1]}
          </a>
        );
      }
      last = m.index + tok.length;
    }
    if (last < text.length) parts.push(text.slice(last));
    return parts;
  };

  while (i < lines.length) {
    const line = lines[i];

    if (line.startsWith("```")) {
      const lang = line.slice(3).trim();
      const buf: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        buf.push(lines[i]);
        i++;
      }
      i++; // closing fence
      out.push(
        <div key={key++} className="my-4">
          {lang && <div className="text-xs text-zinc-500 mb-1">{lang}</div>}
          <pre className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 overflow-x-auto text-xs leading-5">
            <code>{buf.join("\n")}</code>
          </pre>
        </div>
      );
      continue;
    }

    if (line.startsWith("### ")) {
      out.push(<h3 key={key++} className="text-lg font-semibold text-orange-300 mt-6 mb-2">{inline(line.slice(4))}</h3>);
    } else if (line.startsWith("## ")) {
      out.push(<h2 key={key++} className="text-xl font-bold text-orange-400 mt-8 mb-3 border-b border-zinc-800 pb-1">{inline(line.slice(3))}</h2>);
    } else if (line.startsWith("# ")) {
      out.push(<h1 key={key++} className="text-2xl font-bold text-orange-500 mt-8 mb-4">{inline(line.slice(2))}</h1>);
    } else if (/^[-*] /.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^[-*] /.test(lines[i])) {
        items.push(lines[i].slice(2));
        i++;
      }
      out.push(
        <ul key={key++} className="list-disc list-inside space-y-1 my-3 text-zinc-300">
          {items.map((it, n) => <li key={n}>{inline(it)}</li>)}
        </ul>
      );
      continue;
    } else if (/^\d+\. /.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\d+\. /.test(lines[i])) {
        items.push(lines[i].replace(/^\d+\. /, ""));
        i++;
      }
      out.push(
        <ol key={key++} className="list-decimal list-inside space-y-1 my-3 text-zinc-300">
          {items.map((it, n) => <li key={n}>{inline(it)}</li>)}
        </ol>
      );
      continue;
    } else if (line.startsWith("> ")) {
      out.push(<blockquote key={key++} className="border-l-4 border-orange-500 bg-zinc-900 pl-4 pr-3 py-2 my-3 text-zinc-400 rounded-r">{inline(line.slice(2))}</blockquote>);
    } else if (line.trim() === "") {
      // skip blank lines; spacing handled by margins
    } else if (line.trim() === "---") {
      out.push(<hr key={key++} className="border-zinc-800 my-6" />);
    } else {
      out.push(<p key={key++} className="my-2 text-zinc-300">{inline(line)}</p>);
    }
    i++;
  }
  return out;
}

export default function Doc({
  docs,
  slugOverride,
}: {
  docs: { slug: string; label: string }[];
  slugOverride?: string;
}) {
  const { slug: slugFromRoute } = useParams();
  const slug = slugOverride ?? slugFromRoute;
  const [content, setContent] = useState<string | null>(null);
  useEffect(() => {
    setContent(null);
    fetch(`${BASE}docs-content/${slug}.md`)
      .then((r) => (r.ok ? r.text() : Promise.reject(r.status)))
      .then(setContent)
      .catch(() => setContent("# Not found\n\nThis document does not exist yet."));
    window.scrollTo(0, 0);
  }, [slug]);
  // Route-specific title for browser history, tabs and JS-rendering crawlers.
  useEffect(() => {
    const label = docs.find((d) => d.slug === slug)?.label;
    document.title = label
      ? `${label} | FYRwall Documentation`
      : "FYRwall - Open Source Linux Firewall Manager | UFW & iptables Web GUI";
  }, [slug, docs]);

  return (
    <div className="grid md:grid-cols-[220px_1fr] gap-8">
      <aside className="md:sticky md:top-6 self-start">
        <div className="text-xs uppercase tracking-wider text-zinc-500 mb-2 px-2">Documentation</div>
        <ul className="space-y-1 text-sm">
          {docs.map((d) => (
            <li key={d.slug}>
              <NavLink
                to={`/docs/${d.slug}`}
                className={({ isActive }) =>
                  isActive
                    ? "text-orange-400 font-medium bg-zinc-900 rounded block px-2 py-1"
                    : "text-zinc-400 hover:text-orange-400 block px-2 py-1"
                }
              >
                {d.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </aside>
      <article className="min-w-0">
        {content === null ? (
          <div className="text-zinc-500 animate-pulse">Loading...</div>
        ) : (
          renderMarkdown(content)
        )}
      </article>
    </div>
  );
}
