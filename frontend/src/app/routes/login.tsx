import { createFileRoute } from "@tanstack/react-router";

import { LoginPage } from "@/pages/auth/login-page";

type LoginSearch = {
  redirect?: string;
};

export const Route = createFileRoute("/login")({
  validateSearch: (search: Record<string, unknown>): LoginSearch => ({
    redirect: typeof search.redirect === "string" ? search.redirect : undefined,
  }),
  component: LoginPage,
  head: () => ({
    title: "Log in | Holonic Asset",
    meta: [
      {
        name: "description",
        content: "Log in to your Holonic Asset workspace.",
      },
    ],
  }),
});
