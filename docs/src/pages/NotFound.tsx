// In-app 404, mirroring public/404.html styling.
import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <div className="dot-backdrop">
      <div className="mx-auto flex min-h-[60vh] max-w-2xl flex-col items-center px-6 py-20 text-center">
        <p className="eyebrow">Error 404 / Route unknown</p>
        <h1 className="grad-text mt-2 text-[clamp(2.6em,6vw,4em)] font-extrabold tracking-tight">
          Page not found
        </h1>
        <p className="mt-3 text-muted">
          This FYRwall documentation page does not exist. The docs live
          under <code className="inline-code">/docs/</code>, starting at the{" "}
          <Link to="/docs">docs hub</Link>.
        </p>
        <div className="mt-7 flex flex-wrap justify-center gap-3">
          <Link to="/" className="btn btn-primary btn-sm">
            Docs home
          </Link>
          <Link to="/docs/installation" className="btn btn-ghost btn-sm">
            Installation guide
          </Link>
          <a
            href="https://github.com/PotenFYR-Studios/FYRwall/issues"
            className="btn btn-ghost btn-sm"
            target="_blank"
            rel="noopener noreferrer"
          >
            Report an issue
          </a>
        </div>
      </div>
    </div>
  );
}
