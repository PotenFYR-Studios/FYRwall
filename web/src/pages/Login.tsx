import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ShieldCheck, Lock, User, LoaderCircle } from "lucide-react";

import { BlurFade, DotPattern, ShimmerText } from "../components/magicui";
import { sendJSON, useInvalidateMe } from "../lib/auth";

export default function Login() {
  const navigate = useNavigate();
  const invalidateMe = useInvalidateMe();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await sendJSON<{ username: string }>("/api/v1/auth/login", "POST", {
        username,
        password,
      });
      invalidateMe();
      navigate("/", { replace: true });
    } catch (err) {
      setError((err as Error).message || "Sign in failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-zinc-950 px-4" data-testid="login-page">
      <DotPattern className="fill-zinc-700/30 [mask-image:radial-gradient(320px_circle_at_center,white,transparent)]" />
      <BlurFade className="relative w-full max-w-sm">
        <form
          onSubmit={onSubmit}
          className="rounded-xl border border-zinc-800 bg-zinc-900/80 p-6 shadow-2xl shadow-orange-950/20 backdrop-blur"
        >
          <div className="mb-6 flex flex-col items-center gap-2">
            <div className="flex h-12 w-12 items-center justify-center rounded-full border border-fyr-500/30 bg-fyr-500/10">
              <ShieldCheck className="h-6 w-6 text-fyr-500" />
            </div>
            <h1 className="text-xl font-bold">
              <ShimmerText>FYRwall</ShimmerText>
            </h1>
            <p className="text-xs text-zinc-500">Sign in to the management console</p>
          </div>

          {error && (
            <div className="mb-4 rounded border border-red-900 bg-red-950/40 px-3 py-2 text-sm text-red-300" role="alert">
              {error}
            </div>
          )}

          <label className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-500" htmlFor="login-username">
            Username
          </label>
          <div className="relative mb-4">
            <User className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-zinc-600" />
            <input
              id="login-username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              required
              className="w-full rounded border border-zinc-700 bg-zinc-950 py-2 pl-9 pr-3 text-sm text-zinc-100 placeholder-zinc-600 focus:border-fyr-500 focus:outline-none"
              placeholder="admin"
            />
          </div>

          <label className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-500" htmlFor="login-password">
            Password
          </label>
          <div className="relative mb-6">
            <Lock className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-zinc-600" />
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              required
              className="w-full rounded border border-zinc-700 bg-zinc-950 py-2 pl-9 pr-3 text-sm text-zinc-100 placeholder-zinc-600 focus:border-fyr-500 focus:outline-none"
              placeholder="••••••••••••"
            />
          </div>

          <button
            type="submit"
            disabled={busy}
            className="flex w-full items-center justify-center gap-2 rounded bg-fyr-600 py-2 text-sm font-semibold text-white transition hover:bg-fyr-500 disabled:opacity-50"
          >
            {busy && <LoaderCircle className="h-4 w-4 animate-spin" />}
            Sign in
          </button>
        </form>
      </BlurFade>
    </div>
  );
}
