import { redirect } from "@tanstack/react-router";

import { readAuthSession } from "./auth-session.storage";

export function requireAuth(locationHref: string) {
  if (readAuthSession()) return;

  throw redirect({
    to: "/login",
    search: { redirect: locationHref },
    replace: true,
  });
}

export function resolveAuthRedirect(value: string | undefined): string {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return "/projects";
  }
  if (value === "/login" || value.startsWith("/login?")) return "/projects";
  return value;
}
