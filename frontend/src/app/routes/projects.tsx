import { Outlet, createFileRoute } from "@tanstack/react-router";
import { requireAuth } from "@/app/auth-navigation";

export const Route = createFileRoute("/projects")({
  beforeLoad: ({ location }) => requireAuth(location.href),
  component: Outlet,
});
