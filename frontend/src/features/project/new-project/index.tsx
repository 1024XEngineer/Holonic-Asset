import { NewProjectWorkspace } from "./new-project";
import { useNewProjectController } from "./use-new-project-controller";

export function NewProject() {
  const project = useNewProjectController();

  return <NewProjectWorkspace project={project} />;
}
