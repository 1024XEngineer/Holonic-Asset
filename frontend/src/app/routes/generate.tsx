import { createFileRoute } from "@tanstack/react-router";

import { QuickGeneratePage } from "@/pages/quick-generation/quick-generate-page";

export const Route = createFileRoute("/generate")({
  component: QuickGeneratePage,
});
