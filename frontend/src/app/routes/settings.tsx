import { createFileRoute } from "@tanstack/react-router";
import { Settings } from "@/features/settings";
import { requireAuth } from "@/app/auth-navigation";

export const Route = createFileRoute("/settings")({
  beforeLoad: ({ location }) => requireAuth(location.href),
  component: Settings,
});
