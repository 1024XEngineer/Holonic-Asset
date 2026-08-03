import { Outlet, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/projects")({
  validateSearch: (search: Record<string, unknown>) => ({
    project: typeof search.project === "string" ? search.project : undefined,
    q: typeof search.q === "string" ? search.q : undefined,
  }),
  component: Outlet,
});
