import { createFileRoute } from "@tanstack/react-router";

import { EditorPage } from "@/pages/projects/asset/editor-page";

export const Route = createFileRoute("/projects/$projectId/assets/$assetId")({
  component: EditorPage,
});
