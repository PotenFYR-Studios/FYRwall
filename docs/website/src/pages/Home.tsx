export default function Home() {
  return (
    <div className="space-y-10">
      <section className="text-center py-10">
        <h1 className="text-4xl font-bold">
          The <span className="text-orange-500">safe</span> way to manage Linux firewalls
        </h1>
        <p className="mt-4 text-zinc-400 text-lg">
          Free, open-source web GUI for UFW and iptables. Transactional changes with
          automatic rollback. Lockout protection built in. Runs on any distro, any arch.
        </p>
        <div className="mt-6 flex gap-4 justify-center flex-wrap">
          <a
            href="https://github.com/PotenFYR-Studios/FYRwall#install"
            className="rounded-lg bg-orange-600 hover:bg-orange-500 px-6 py-3 font-semibold"
          >
            Get Started
          </a>
          <a
            href="https://github.com/PotenFYR-Studios/FYRwall"
            className="rounded-lg border border-zinc-700 px-6 py-3 font-semibold hover:bg-zinc-900"
          >
            View Source
          </a>
        </div>
        <div className="mt-8 text-sm text-zinc-500">
          <code className="bg-zinc-900 rounded px-3 py-2">curl -fsSL https://potenfyr-studios.github.io/FYRwall/install.sh | sudo sh</code>
        </div>
      </section>

      <section className="grid md:grid-cols-3 gap-4">
        {[
          ["Never locked out", "Every risky change is blocked or auto-rolled back with restore points."],
          ["Unprivileged by design", "The web server never runs as root. Typed agent socket, zero shell."],
          ["Any Linux, any arch", "Debian to Alpine, amd64 to riscv64. Offline-capable, no telemetry."],
        ].map(([title, body]) => (
          <div key={title} className="rounded-lg border border-zinc-800 bg-zinc-900 p-5">
            <h2 className="font-semibold text-orange-400">{title}</h2>
            <p className="mt-2 text-sm text-zinc-400">{body}</p>
          </div>
        ))}
      </section>
    </div>
  );
}
