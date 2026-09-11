import { Routes, Route, NavLink } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Rules from "./pages/Rules";
import Health from "./pages/Health";
import Notifications from "./pages/Notifications";
import About from "./pages/About";
import { HealthBadge } from "./components/HealthBadge";

const nav = [
  { to: "/", label: "Dashboard" },
  { to: "/firewall/rules", label: "Rules" },
  { to: "/system/health", label: "Health" },
  { to: "/notifications", label: "Notifications" },
  { to: "/about", label: "About" },
];

export default function App() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-3 flex items-center gap-6">
        <span className="font-bold text-fyr-500 text-lg">FYRwall</span>
        <nav className="flex gap-4">
          {nav.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              className={({ isActive }) =>
                `text-sm px-2 py-1 rounded ${isActive ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-white"}`
              }
            >
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="ml-auto">
          <HealthBadge />
        </div>
      </header>
      <main className="p-6 max-w-7xl mx-auto">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/firewall/rules" element={<Rules />} />
          <Route path="/system/health" element={<Health />} />
          <Route path="/notifications" element={<Notifications />} />
          <Route path="/about" element={<About />} />
        </Routes>
      </main>
    </div>
  );
}
