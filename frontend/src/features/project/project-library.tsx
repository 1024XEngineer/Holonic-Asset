import { useProjectLibrary } from "./state/use-project-library";
import { ProjectLibraryWorkspace } from "./project-library-workspace";

export function ProjectLibrary({ projectId }: { projectId?: string }) {
  const library = useProjectLibrary(projectId);

  return <ProjectLibraryWorkspace library={library} />;
}
