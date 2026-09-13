import { Routes, Route, NavLink, Navigate, useLocation } from "react-router-dom";
import { LogOut, ShieldCheck } from "lucide-react";

import Dashboard from "./pages/Dashboard";
import Rules from "./pages/Rules";
import Health from "./pages/Health";
import Notifications from "./pages/Notifications";
import About from "./pages/About";
import Login from "./pages/Login";
import Setup from "./pages/Setup";
import Fleet from "./pages/Fleet";
import { HealthBadge } from "./components/HealthBadge";
import { BlurFade, DotPattern, ShimmerText } from "./components/magicui";
import { sendJSON, useInvalidateMe, useMe, useSetupNeeded } from "./lib/auth";

const nav = [
  { to: "/", label: "Dashboard" },
  { to: "/firewall/rules", label: "Rules" },
  { to: "/fleet", label: "Fleet" },
  { to: "/system/health", label: "Health" },
  { to: "/notifications", label: "Notifications" },
  { to: "/about", label: "About" },
];

function Shell({ children }: { children: React.ReactNode }) {
  const invalidateMe = useInvalidateMe();
  const me = useMe();

  async function logout() {
    try {
      await sendJSON("/api/v1/auth/logout", "POST");
    } finally {
      invalidateMe();
    }
  }

  return (
    <div className="relative min-h-screen bg-zinc-950 text-zinc-100">
      <div aria-hidden="true" className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-80 overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(249,115,22,0.16),transparent_65%)] blur-2xl" />
        <DotPattern className="h-full w-full fill-amber-500/20 [mask-image:radial-gradient(70%_60%_at_50%_0%,white,transparent)]" />
      </div>
      <header className="sticky top-0 z-10 border-b border-zinc-800 bg-zinc-950/90 px-6 py-3 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center gap-6">
          <span className="flex items-center gap-2 font-bold text-lg">
            <ShieldCheck className="h-5 w-5 text-fyr-500" />
            <ShimmerText>FYRwall</ShimmerText>
          </span>
          <nav className="flex gap-1">
            {nav.map((n) => (
              <NavLink
                key={n.to}
                to={n.to}
                className={({ isActive }) =>
                  `rounded px-2.5 py-1 text-sm transition ${isActive ? "bg-zinc-800 text-white" : "text-zinc-400 hover:bg-zinc-900 hover:text-white"}`
                }
              >
                {n.label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-3">
            <HealthBadge />
            {me.data?.authenticated && (
              <span className="hidden text-xs text-zinc-500 sm:inline">
                {me.data.username}
              </span>
            )}
            <button
              onClick={logout}
              title="Sign out"
              data-testid="logout"
              className="flex items-center gap-1.5 rounded border border-zinc-800 px-2.5 py-1 text-xs text-zinc-400 transition hover:border-zinc-700 hover:text-white"
            >
              <LogOut className="h-3.5 w-3.5" />
              Sign out
            </button>
          </div>
        </div>
      </header>
      <BlurFade duration={0.5}>
        <main className="mx-auto max-w-7xl p-6">{children}</main>
      </BlurFade>
    </div>
  );
}

/** Gate: setup wizard on fresh installs, login when signed out. */
function Guard({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const setup = useSetupNeeded();
  const location = useLocation();

  if (me.isLoading || setup.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950 text-sm text-zinc-500">
        Loading…
      </div>
    );
  }
  if (setup.data?.setup_needed) return <Navigate to="/setup" replace />;
  if (!me.data?.authenticated)
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  return <Shell>{children}</Shell>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/setup" element={<Setup />} />
      <Route
        path="*"
        element={
          <Guard>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/firewall/rules" element={<Rules />} />
              <Route path="/fleet" element={<Fleet />} />
              <Route path="/system/health" element={<Health />} />
              <Route path="/notifications" element={<Notifications />} />
              <Route path="/about" element={<About />} />
            </Routes>
          </Guard>
        }
      />
    </Routes>
  );
}
