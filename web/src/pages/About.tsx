export default function About() {
  const restrictions: { ok: boolean; text: string }[] = [
    { ok: true, text: "Personal and internal business use" },
    { ok: true, text: "Self-hosting for your company or clients" },
    { ok: true, text: "Modifying and redistributing under the same license" },
    { ok: true, text: "Embedding FYRwall as a feature of a larger paid product" },
    { ok: true, text: "Building paid services around FYRwall" },
    { ok: false, text: "Selling FYRwall itself (or copies) for a fee" },
    { ok: false, text: "Paid managed hosting of FYRwall alone" },
    { ok: false, text: "Paid support whose value derives entirely from FYRwall" },
    { ok: false, text: "Removing LICENSE or attribution notices" },
  ];

  return (
    <div className="space-y-4 max-w-3xl">
      <h2 className="text-lg font-semibold">About FYRwall</h2>

      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-4">
        <p className="text-sm text-zinc-400">
          FYRwall is free, open-source firewall management for Linux, built by
          PotenFYR Studios. Licensed under{" "}
          <span className="text-zinc-100 font-medium">
            Apache-2.0 WITH Commons Clause v1.0
          </span>
          .
        </p>
      </div>

      <div className="rounded-lg border border-zinc-800 bg-zinc-900 p-4" data-testid="license-restrictions">
        <h3 className="font-semibold text-orange-400">License and restrictions</h3>
        <p className="mt-1 text-sm text-zinc-500">
          The Commons Clause draws one bright line: FYRwall stays free and
          open, and nobody may resell it as-is. Everything else - including
          commercial use as a component of your own product - is welcome.
        </p>
        <ul className="mt-3 space-y-1.5 text-sm">
          {restrictions.map((r) => (
            <li key={r.text} className="flex items-start gap-2">
              <span className={r.ok ? "text-green-400" : "text-red-400"}>
                {r.ok ? "Allowed:" : "Not allowed:"}
              </span>
              <span className={r.ok ? "text-zinc-300" : "text-zinc-400"}>
                {r.text}
              </span>
            </li>
          ))}
        </ul>
        <p className="mt-3 text-xs text-zinc-600">
          Full legal text: the LICENSE file in the installation, or{" "}
          <a
            className="text-orange-400 hover:underline"
            href="https://github.com/PotenFYR-Studios/FYRwall/blob/main/LICENSE"
            target="_blank"
            rel="noopener noreferrer"
          >
            view online
          </a>
          . Commercial exception questions:{" "}
          <a className="text-orange-400 hover:underline" href="https://www.potenfyr.in/" target="_blank" rel="noopener noreferrer">
            potenfyr.in
          </a>
          .
        </p>
      </div>
    </div>
  );
}
