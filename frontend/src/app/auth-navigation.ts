import { redirect } from "@tanstack/react-router";

import { readAuthSession } from "@/features/auth/session";

export function requireAuth(redirectTo: string) {
  if (readAuthSession()) return;
  throw redirect({
    to: "/login",
    search: { redirect: redirectTo },
    replace: true,
  });
}

export function resolveAuthRedirect(value: string | undefined): string {
  if (
    !value ||
    !value.startsWith("/") ||
    value.startsWith("//") ||
    value.includes("\\") ||
    hasControlCharacter(value)
  ) {
    return "/projects";
  }
  if (value === "/login" || value.startsWith("/login?")) return "/projects";
  return value;
}

function hasControlCharacter(value: string) {
  return Array.from(value).some((character) => {
    const code = character.charCodeAt(0);
    return code <= 0x1f || code === 0x7f;
  });
}
