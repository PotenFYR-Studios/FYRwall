import { FormEvent, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Rocket, LoaderCircle, ShieldCheck, Check, X } from "lucide-react";

import { BlurFade, BorderBeam, DotPattern, ShimmerText } from "../components/magicui";
import { sendJSON, useInvalidateMe, useSetupNeeded } from "../lib/auth";

const MIN_LEN = 14;

type Rule = { label: string; ok: boolean };
function policyRules(pw: string, confirm: string): Rule[] {
  return [
    { label: `At least ${MIN_LEN} characters`, ok: pw.length >= MIN_LEN },
    { label: "Upper and lower case letter", ok: /[a-z]/.test(pw) && /[A-Z]/.test(pw) },
    { label: "At least one digit", ok: /\d/.test(pw) },
    { label: "At least one special character", ok: /[^A-Za-z0-9]/.test(pw) },
    { label: "No three identical characters in a row", ok: pw !== "" && !/(.)\1\1/.test(pw) },
    { label: "Passwords match", ok: confirm !== "" && pw === confirm },
  ];
}

/**
 * One-time super admin setup wizard. Shown when /api/v1/setup/status
 * reports setup_needed: true (fresh install, e.g. a Docker container).
 * The account + password are created here; the CLI never asks for one.
 */
export default function Setup() {
  const navigate = useNavigate();
  const invalidateMe = useInvalidateMe();
  const { data: setup } = useSetupNeeded();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const rules = policyRules(password, confirm);
  const allOk = rules.every((r) => r.ok);

  // Already set up (or racing another tab): bounce to login.
  useEffect(() => {
    if (setup && !setup.setup_needed) navigate("/login", { replace: true });
  }, [setup, navigate]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!allOk) return;
    setError("");
    setBusy(true);
    try {
      await sendJSON("/api/v1/setup/super-admin", "POST", {
        username,
        password,
        confirm_password: confirm,
      });
      // The setup response carries no session; sign in once with the new
      // credentials so the user lands inside the console.
      await sendJSON("/api/v1/auth/login", "POST", { username, password });
      invalidateMe();
      navigate("/", { replace: true });
    } catch (err) {
      setError((err as Error).message || "Setup failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-zinc-950 px-4" data-testid="setup-page">
      <DotPattern className="fill-zinc-700/30 [mask-image:radial-gradient(320px_circle_at_center,white,transparent)]" />
      <BlurFade className="relative w-full max-w-md">
        <form
          onSubmit={onSubmit}
          className="relative overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/80 p-6 shadow-2xl shadow-orange-950/20 backdrop-blur"
        >
          <BorderBeam size={60} duration={8} />
          <div className="mb-6 flex flex-col items-center gap-2">
            <div className="flex h-12 w-12 items-center justify-center rounded-full border border-fyr-500/30 bg-fyr-500/10">
              <Rocket className="h-6 w-6 text-fyr-500" />
            </div>
            <h1 className="text-xl font-bold">
              <ShimmerText>Welcome to FYRwall</ShimmerText>
            </h1>
            <p className="text-center text-xs text-zinc-500">
              One-time setup: create the super admin account for this
              installation. This screen appears only once.
            </p>
          </div>

          {error && (
            <div className="mb-4 rounded border border-red-900 bg-red-950/40 px-3 py-2 text-sm text-red-300" role="alert">
              {error}
            </div>
          )}

          <label className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-500" htmlFor="setup-username">
            Admin username
          </label>
          <input
            id="setup-username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            required
            minLength={3}
            className="mb-4 w-full rounded border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:border-fyr-500 focus:outline-none"
            placeholder="admin"
          />

          <label className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-500" htmlFor="setup-password">
            Password
          </label>
          <input
            id="setup-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
            required
            className="mb-1 w-full rounded border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:border-fyr-500 focus:outline-none"
            placeholder="Choose a strong password"
          />
          <input
            id="setup-password-confirm"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
            required
            className="mb-4 mt-3 w-full rounded border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:border-fyr-500 focus:outline-none"
            placeholder="Repeat password"
          />

          <ul className="mb-5 space-y-1.5" data-testid="password-policy">
            {rules.map((r) => (
              <li key={r.label} className={`flex items-center gap-2 text-xs ${r.ok ? "text-green-400" : "text-zinc-500"}`}>
                {r.ok ? <Check className="h-3.5 w-3.5" /> : <X className="h-3.5 w-3.5" />}
                {r.label}
              </li>
            ))}
          </ul>

          <button
            type="submit"
            disabled={busy || !allOk}
            className="flex w-full items-center justify-center gap-2 rounded bg-fyr-600 py-2 text-sm font-semibold text-white transition hover:bg-fyr-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <ShieldCheck className="h-4 w-4" />}
            Create super admin
          </button>
        </form>
      </BlurFade>
    </div>
  );
}
