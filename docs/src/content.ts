// Docs manifest: the single source of truth for sidebar order, page
// titles, meta descriptions and the markdown bodies (imported at build
// time from docs/content/*.md - the same files the README links to).

export interface DocMeta {
  slug: string;
  title: string;
  description: string;
  group: string;
}

export const DOC_GROUPS = ["Get started", "Core topics", "Reference"] as const;

export const DOCS: DocMeta[] = [
  {
    slug: "installation",
    title: "Installation",
    description:
      "Install FYRwall with the one-liner script, review-first, or from a release tarball. The installer never touches your firewall rules.",
    group: "Get started",
  },
  {
    slug: "docker",
    title: "Docker",
    description:
      "Run the FYRwall server in a container and bridge a host-networked agent so the container can reach the real host firewall.",
    group: "Get started",
  },
  {
    slug: "configuration",
    title: "Configuration",
    description:
      "Every FYRwall config key, FYRWALL_* environment overrides and config validation in one reference.",
    group: "Get started",
  },
  {
    slug: "updating",
    title: "Updating",
    description:
      "Update FYRwall safely: database backup, restore point, verified download, migrations, health checks. Config is never overwritten.",
    group: "Get started",
  },
  {
    slug: "architecture",
    title: "Architecture",
    description:
      "How FYRwall is built: unprivileged server, typed Unix-socket agent, backend adapters, ownership detection and storage.",
    group: "Core topics",
  },
  {
    slug: "safety",
    title: "Safety and Restore Points",
    description:
      "The transactional apply pipeline, automatic restore points, SHA-256 verification, the safe apply timer and rollback guarantees.",
    group: "Core topics",
  },
  {
    slug: "security",
    title: "Security Model",
    description:
      "Privilege separation, Argon2id auth, CSRF, RBAC, encrypted agent logs, secret handling and what extensions can never do.",
    group: "Core topics",
  },
  {
    slug: "operation",
    title: "Operation",
    description:
      "Daily workflow in the web UI, CLI essentials, tray and desktop integration, users, roles and notifications.",
    group: "Core topics",
  },
  {
    slug: "extensions",
    title: "Extensions",
    description:
      "Capability-scoped, declarative extensions: dashboard widgets, notification channels, probes, rule templates and sandboxed panels.",
    group: "Reference",
  },
  {
    slug: "extensions-guide",
    title: "Extensions Guide",
    description:
      "Full manifest reference for writing FYRwall extensions, with the hard rules the installer enforces.",
    group: "Reference",
  },
  {
    slug: "troubleshooting",
    title: "Troubleshooting",
    description:
      "Fixes for an unreachable UI, backend conflicts, SSH lockout, offline agents, disk usage and password recovery.",
    group: "Reference",
  },
  {
    slug: "faq",
    title: "FAQ",
    description:
      "Common questions: licensing, supported distros, fleet management, privacy, commercial use and clean removal.",
    group: "Reference",
  },
  {
    slug: "license",
    title: "License",
    description:
      "FYRwall is licensed under Apache-2.0 with Commons Clause. What is allowed, what is not, and why.",
    group: "Reference",
  },
];

const bodies = import.meta.glob("../content/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

export function docBody(slug: string): string | undefined {
  return bodies[`../content/${slug}.md`];
}

export function docBySlug(slug: string | undefined): DocMeta | undefined {
  return DOCS.find((d) => d.slug === slug);
}

export const SITE_URL = "https://fyrwall.docs.potenfyr.in";

/** Per-route SEO meta used by both the prerenderer and the router. */
export function routeMeta(
  pathname: string
): { title: string; description: string } {
  const clean = pathname.replace(/\/+$/, "") || "/";
  if (clean === "/") {
    return {
      title: "FYRwall - Linux Firewall Manager for UFW & iptables",
      description:
        "Open-source web GUI for UFW and iptables. Transactional changes with rollback, restore points, RBAC and multi-host agents. Self-hosted on any Linux distro.",
    };
  }
  if (clean === "/docs") {
    return {
      title: "Documentation | FYRwall",
      description:
        "FYRwall documentation: installation, Docker, configuration, architecture, safety and restore points, security model, operation, extensions, troubleshooting and licensing.",
    };
  }
  if (clean === "/examples") {
    return {
      title: "Examples | FYRwall",
      description:
        "Copy-paste FYRwall examples: the install one-liner, first admin, config files, Docker compose, CLI cookbook, extension manifests and the API surface.",
    };
  }
  if (clean === "/about") {
    return {
      title: "About | FYRwall",
      description:
        "Why FYRwall exists, the core guarantees, and PotenFYR Studios - the org behind the safe Linux firewall manager.",
    };
  }
  const slug = clean.replace(/^\/docs\//, "");
  const meta = docBySlug(slug);
  if (meta) {
    return { title: `${meta.title} | FYRwall Documentation`, description: meta.description };
  }
  return {
    title: "Page not found | FYRwall",
    description: "This FYRwall documentation page could not be found.",
  };
}
