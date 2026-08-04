import { NewProjectWorkspace } from "./new-project-workspace";
import { useNewProjectController } from "./use-new-project-controller";

export function NewProject() {
  return <NewProjectWorkspace project={useNewProjectController()} />;
}
