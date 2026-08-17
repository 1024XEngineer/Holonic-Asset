import { AppHeader } from "@/components/layouts/app-header";
import { ProjectWorkspaceLayout } from "@/components/layouts/project-workspace-layout";
import { AssetLibrary } from "@/features/assets";

import { ProjectSidebar } from "./project-sidebar";
import type { ProjectLibraryController } from "./state/use-project-library";

export function ProjectLibraryWorkspace({
  library,
}: {
  library: ProjectLibraryController;
}) {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <ProjectWorkspaceLayout
        sidebar={<ProjectSidebar library={library.project} />}
      >
        <AssetLibrary
          isProjectLoading={library.project.isLoading}
          project={library.project.current}
        />
      </ProjectWorkspaceLayout>
    </div>
  );
}
