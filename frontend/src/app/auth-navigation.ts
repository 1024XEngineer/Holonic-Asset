import { redirect } from "@tanstack/react-router";

import { readAuthSession } from "@/model/auth";

export function requireAuth(redirectTo: string) {
  if (readAuthSession()) return;
  throw redirect({
    to: "/login",
    search: { redirect: redirectTo },
    replace: true,
  });
}
