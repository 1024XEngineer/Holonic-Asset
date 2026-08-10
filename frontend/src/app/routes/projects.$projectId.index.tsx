import { createFileRoute } from "@tanstack/react-router";

import { ProjectLibraryPage } from "@/pages/projects/project-library-page";

export const Route = createFileRoute("/projects/$projectId/")({
  component: ProjectRoute,
});

function ProjectRoute() {
  const { projectId } = Route.useParams();

  return <ProjectLibraryPage projectId={projectId} />;
}
