// Lightweight markdown renderer for docs/content/*.md. Handles what the
// content actually uses: headings, 4-space indented code (the repo
// convention), fenced code, pipe tables, lists, quotes, hr, and inline
// code/bold/italic/links. React nodes only - never dangerouslySetInnerHTML.
// Emits heading ids so the TOC and palette can anchor.
import { useState } from "react";

export interface Heading {
  id: string;
  text: string;
  level: 2 | 3 | 4;
}

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function CopyButton({ text }: { text: string }) {
  const [ok, setOk] = useState(false);
  return (
    <button
      type="button"
      className={`copy-btn${ok ? " ok" : ""}`}
      onClick={() => {
        navigator.clipboard?.writeText(text).then(
          () => {
            setOk(true);
            setTimeout(() => setOk(false), 1400);
          },
          () => undefined
        );
      }}
    >
      {ok ? "Copied!" : "Copy"}
    </button>
  );
}

function CodeBlock({ code, lang }: { code: string; lang?: string }) {
  return (
    <pre data-lang={lang ?? ""}>
      <code>{code}</code>
      <CopyButton text={code} />
    </pre>
  );
}

const INLINE_RE = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\*[^*]+\*)|(\[[^\]]+\]\([^)\s]+\))/g;

function inline(text: string, keyBase: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  let last = 0;
  let k = 0;
  let m: RegExpExecArray | null;
  INLINE_RE.lastIndex = 0;
  while ((m = INLINE_RE.exec(text)) !== null) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    const tok = m[0];
    if (tok.startsWith("`")) {
      parts.push(<code key={`${keyBase}-${k++}`}>{tok.slice(1, -1)}</code>);
    } else if (tok.startsWith("**")) {
      parts.push(<strong key={`${keyBase}-${k++}`}>{tok.slice(2, -2)}</strong>);
    } else if (tok.startsWith("*")) {
      parts.push(<em key={`${keyBase}-${k++}`}>{tok.slice(1, -1)}</em>);
    } else {
      const lm = /\[([^\]]+)\]\(([^)\s]+)\)/.exec(tok)!;
      const href = lm[2];
      const external = /^https?:\/\//.test(href);
      parts.push(
        <a
          key={`${keyBase}-${k++}`}
          href={href}
          target={external ? "_blank" : undefined}
          rel={external ? "noopener noreferrer" : undefined}
        >
          {lm[1]}
        </a>
      );
    }
    last = m.index + tok.length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts;
}

/** Render markdown to React. Returns nodes plus extracted headings for the TOC. */
export function renderMarkdown(
  md: string
): { nodes: React.ReactNode[]; headings: Heading[] } {
  const nodes: React.ReactNode[] = [];
  const headings: Heading[] = [];
  const lines = md.split("\n");
  let i = 0;
  let key = 0;

  while (i < lines.length) {
    const line = lines[i];

    // fenced code block
    if (line.startsWith("```")) {
      const lang = line.slice(3).trim() || undefined;
      const buf: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        buf.push(lines[i]);
        i++;
      }
      i++; // closing fence
      nodes.push(<CodeBlock key={key++} code={buf.join("\n")} lang={lang} />);
      continue;
    }

    // 4-space indented code block (repo convention, no fences)
    if (/^ {4}\S/.test(line)) {
      const buf: string[] = [];
      while (i < lines.length && (/^ {4}/.test(lines[i]) || lines[i].trim() === "")) {
        if (lines[i].trim() === "" && i + 1 < lines.length && !/^ {4}/.test(lines[i + 1])) break;
        buf.push(lines[i].replace(/^ {4}/, ""));
        i++;
      }
      nodes.push(<CodeBlock key={key++} code={buf.join("\n").replace(/\n+$/, "")} />);
      continue;
    }

    // pipe table: header row, |---| separator, body rows
    if (
      line.includes(" | ") &&
      i + 1 < lines.length &&
      /^\s*\|?[\s:-]*-[\s|:-]*$/.test(lines[i + 1]) &&
      lines[i + 1].includes("-")
    ) {
      const cells = (row: string) =>
        row.replace(/^\s*\|/, "").replace(/\|\s*$/, "").split("|").map((c) => c.trim());
      const head = cells(line);
      i += 2;
      const rows: string[][] = [];
      while (i < lines.length && lines[i].includes("|")) {
        rows.push(cells(lines[i]));
        i++;
      }
      nodes.push(
        <div key={key++} className="table-scroll panel-scroll">
          <table className="doc-table">
            <thead>
              <tr>{head.map((h, n) => <th key={n}>{inline(h, `${key}-h${n}`)}</th>)}</tr>
            </thead>
            <tbody>
              {rows.map((r, n) => (
                <tr key={n}>
                  {r.map((c, m2) => <td key={m2}>{inline(c, `${key}-r${n}c${m2}`)}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
      continue;
    }

    if (line.startsWith("#### ")) {
      const text = line.slice(5).trim();
      headings.push({ id: slugify(text), text, level: 4 });
      nodes.push(
        <h4 key={key++} id={slugify(text)}>
          {inline(text, `h4-${key}`)}
        </h4>
      );
    } else if (line.startsWith("### ")) {
      const text = line.slice(4).trim();
      headings.push({ id: slugify(text), text, level: 3 });
      nodes.push(
        <h3 key={key++} id={slugify(text)}>
          {inline(text, `h3-${key}`)}
        </h3>
      );
    } else if (line.startsWith("## ")) {
      const text = line.slice(3).trim();
      headings.push({ id: slugify(text), text, level: 2 });
      nodes.push(
        <h2 key={key++} id={slugify(text)}>
          {inline(text, `h2-${key}`)}
        </h2>
      );
    } else if (line.startsWith("# ")) {
      const text = line.slice(2).trim();
      nodes.push(
        <h1 key={key++}>
          {inline(text, `h1-${key}`)}
        </h1>
      );
    } else if (/^[-*] /.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^[-*] /.test(lines[i])) {
        items.push(lines[i].slice(2));
        i++;
      }
      nodes.push(
        <ul key={key++}>
          {items.map((it, n) => (
            <li key={n}>{inline(it, `li-${key}-${n}`)}</li>
          ))}
        </ul>
      );
      continue;
    } else if (/^\d+\. /.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\d+\. /.test(lines[i])) {
        items.push(lines[i].replace(/^\d+\. /, ""));
        i++;
      }
      nodes.push(
        <ol key={key++}>
          {items.map((it, n) => (
            <li key={n}>{inline(it, `oli-${key}-${n}`)}</li>
          ))}
        </ol>
      );
      continue;
    } else if (line.startsWith("> ")) {
      nodes.push(
        <blockquote key={key++}>{inline(line.slice(2), `bq-${key}`)}</blockquote>
      );
    } else if (line.trim() === "---") {
      nodes.push(<hr key={key++} />);
    } else if (line.trim() === "") {
      // blank: spacing is handled by margins
    } else {
      nodes.push(<p key={key++}>{inline(line, `p-${key}`)}</p>);
    }
    i++;
  }

  return { nodes, headings };
}
