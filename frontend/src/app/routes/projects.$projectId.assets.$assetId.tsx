import { createFileRoute } from "@tanstack/react-router";

import { recordQueryOptions } from "@/model";
import { AssetEditorPage } from "@/pages/projects/asset/asset-editor-page";

export const Route = createFileRoute("/projects/$projectId/assets/$assetId")({
  loader: ({ context: { queryClient }, params: { assetId, projectId } }) =>
    queryClient.ensureQueryData(recordQueryOptions(projectId, assetId)),
  component: AssetEditorPage,
});
