import { ProjectLibraryWorkspace } from "./project-library-workspace";
import { useProjectLibrary } from "./state/use-project-library";

export function ProjectLibrary() {
  return <ProjectLibraryWorkspace library={useProjectLibrary()} />;
}
