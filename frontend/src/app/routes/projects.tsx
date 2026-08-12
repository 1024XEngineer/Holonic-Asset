import { Outlet, createFileRoute } from "@tanstack/react-router";
import { requireAuth } from "@/model/auth";

export const Route = createFileRoute("/projects")({
  beforeLoad: ({ location }) => requireAuth(location.href),
  component: Outlet,
});
