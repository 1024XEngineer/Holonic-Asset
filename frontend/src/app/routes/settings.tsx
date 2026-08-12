import { createFileRoute } from "@tanstack/react-router";
import { Settings } from "@/features/settings";
import { requireAuth } from "@/model/auth";

export const Route = createFileRoute("/settings")({
  beforeLoad: ({ location }) => requireAuth(location.href),
  component: Settings,
});
