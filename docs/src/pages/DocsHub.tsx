// Docs hub (SPEC 6): max-w-5xl card grid of every docs page, grouped.
import { Link } from "react-router-dom";
import { ArrowUpRight, FileText } from "lucide-react";
import { DOCS, DOC_GROUPS } from "../content";

export default function DocsHub() {
  return (
    <div className="dot-backdrop">
      <div className="relative mx-auto max-w-5xl px-6 pb-20 pt-14">
        <p className="eyebrow">
          <span className="eyebrow-accent">FYRwall</span> docs / v0.1.0
        </p>
        <h1 className="grad-text mt-3 text-[clamp(2.2em,5vw,3.2em)] font-extrabold leading-[1.08] tracking-tight">
          FYRwall Documentation
        </h1>
        <p className="mt-4 max-w-2xl text-[1.04em] text-muted">
          Everything about installing, configuring and operating the safe
          Linux firewall manager - from the first one-liner to extension
          manifests.
        </p>

        {DOC_GROUPS.map((group) => (
          <section key={group} className="mt-12">
            <div className="side-title" style={{ paddingLeft: 0 }}>
              {group}
            </div>
            <div className="mt-4 grid gap-[14px] sm:grid-cols-2 lg:grid-cols-3">
              {DOCS.filter((d) => d.group === group).map((d) => (
                <Link key={d.slug} to={`/docs/${d.slug}`} className="doc-card">
                  <div className="flex items-center justify-between">
                    <span className="card-icon">
                      <FileText className="h-5 w-5 text-brand-violet" />
                    </span>
                    <ArrowUpRight className="card-arrow h-4 w-4" />
                  </div>
                  <span className="card-title mt-2">{d.title}</span>
                  <span className="card-desc">{d.description}</span>
                </Link>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
