import { useQuery, useQueryClient } from "@tanstack/react-query";

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: "same-origin" });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    const msg = body?.error?.message ?? res.statusText;
    throw new Error(msg);
  }
  return res.json();
}

export { fetchJSON };

export type Me = {
  authenticated: boolean;
  username?: string;
  role?: string;
  csrf_token?: string;
};

export function getCSRFToken(): string {
  return document.documentElement.dataset.csrf ?? "";
}

function setCSRFToken(tok: string | undefined) {
  if (tok) document.documentElement.dataset.csrf = tok;
}

/** POST/PUT/DELETE helper that attaches the session CSRF header. */
export async function sendJSON<T>(
  url: string,
  method: "POST" | "PUT" | "PATCH" | "DELETE",
  body?: unknown,
): Promise<T> {
  const res = await fetch(url, {
    method,
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/json",
      "X-FYRwall-CSRF": getCSRFToken(),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const msg = (data as { error?: { message?: string } })?.error?.message ?? res.statusText;
    throw new Error(msg);
  }
  return data as T;
}

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: async () => {
      const me = await fetchJSON<Me>("/api/v1/auth/me");
      setCSRFToken(me.csrf_token);
      return me;
    },
    staleTime: 60_000,
  });
}

export function useSetupNeeded() {
  return useQuery({
    queryKey: ["setup-status"],
    queryFn: () => fetchJSON<{ setup_needed: boolean }>("/api/v1/setup/status"),
    staleTime: 30_000,
  });
}

export function useInvalidateMe() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: ["me"] });
}
