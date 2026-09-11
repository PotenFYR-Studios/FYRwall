import { useParams, Link } from "react-router-dom";
import { useEffect, useState } from "react";

const BASE = import.meta.env.BASE_URL ?? "/";

export default function Doc({ docs }: { docs: { slug: string; label: string }[] }) {
  const { slug } = useParams();
  const [content, setContent] = useState("Loading...");
  useEffect(() => {
    fetch(`${BASE}docs-content/${slug}.md`)
      .then((r) => (r.ok ? r.text() : Promise.reject(r.status)))
      .then(setContent)
      .catch(() => setContent("# Not found\n\nThis document does not exist yet."));
    window.scrollTo(0, 0);
  }, [slug]);
  return (
    <div className="grid md:grid-cols-[200px_1fr] gap-8">
      <aside>
        <ul className="space-y-1 text-sm">
          {docs.map((d) => (
            <li key={d.slug}>
              <Link
                to={`/docs/${d.slug}`}
                className="text-zinc-400 hover:text-orange-400 block px-2 py-1"
              >
                {d.label}
              </Link>
            </li>
          ))}
        </ul>
      </aside>
      <article className="prose-invert whitespace-pre-wrap font-mono text-sm leading-6">
        {content}
      </article>
    </div>
  );
}
