import { Outlet, useLocation, createFileRoute } from "@tanstack/react-router";

import { ProjectLibraryPage } from "@/pages/projects/project-library-page";

export const Route = createFileRoute("/projects/$projectId")({
  component: ProjectRoute,
});

function ProjectRoute() {
  const { projectId } = Route.useParams();
  const { pathname } = useLocation();
  const isCreateAssetRoute = pathname.startsWith(
    `/projects/${projectId}/assets/new/`,
  );

  return isCreateAssetRoute ? (
    <Outlet />
  ) : (
    <ProjectLibraryPage projectId={projectId} />
  );
}
