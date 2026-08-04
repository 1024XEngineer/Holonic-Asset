import { useProjectLibrary } from "./state/use-project-library";
import { ProjectLibraryWorkspace } from "./project-library-workspace";

export function ProjectLibrary() {
  const library = useProjectLibrary();

  return <ProjectLibraryWorkspace library={library} />;
}
