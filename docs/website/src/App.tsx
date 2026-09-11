import { Routes, Route, NavLink } from "react-router-dom";
import Home from "./pages/Home";
import Doc from "./pages/Doc";

const docs = [
  { slug: "installation", label: "Installation" },
  { slug: "operation", label: "Operation" },
  { slug: "security", label: "Security" },
  { slug: "architecture", label: "Architecture" },
  { slug: "extensions", label: "Extensions" },
  { slug: "updating", label: "Updating" },
  { slug: "troubleshooting", label: "Troubleshooting" },
  { slug: "faq", label: "FAQ" },
  { slug: "license", label: "License" },
];

export default function App() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4 flex items-center gap-8">
        <a href="#/" className="font-bold text-orange-500 text-xl">FYRwall</a>
        <nav className="flex gap-4 flex-wrap">
          {docs.map((d) => (
            <NavLink
              key={d.slug}
              to={`/docs/${d.slug}`}
              className={({ isActive }) =>
                `text-sm ${isActive ? "text-orange-400" : "text-zinc-400 hover:text-white"}`
              }
            >
              {d.label}
            </NavLink>
          ))}
        </nav>
        <a
          href="https://github.com/PotenFYR-Studios/FYRwall"
          className="ml-auto text-sm text-zinc-400 hover:text-white"
          rel="noopener noreferrer"
        >
          GitHub
        </a>
      </header>
      <main className="p-6 max-w-4xl mx-auto">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/docs/:slug" element={<Doc docs={docs} />} />
        </Routes>
      </main>
      <footer className="border-t border-zinc-900 p-6 text-center text-sm text-zinc-600">
        Apache-2.0 with Commons Clause - Built by PotenFYR Studios
      </footer>
    </div>
  );
}
