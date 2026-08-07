import { createFileRoute } from "@tanstack/react-router";

import { CreateAssetPage } from "@/pages/projects/create-asset-page";

export const Route = createFileRoute("/projects/$projectId/assets/new/$kind")({
  component: CreateAssetRoute,
});

function CreateAssetRoute() {
  const { projectId, kind } = Route.useParams();

  return <CreateAssetPage projectId={projectId} rawKind={kind} />;
}
