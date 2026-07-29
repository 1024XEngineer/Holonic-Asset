import { useNavigate } from "@tanstack/react-router";

import { useCreateProjectMutation } from "@/api";
import { NewProjectScreen } from "@/features/project";

export function NewProjectPage() {
  const navigate = useNavigate({ from: "/projects/new" });
  const { mutateAsync: createProject } = useCreateProjectMutation();

  return (
    <div className="min-h-screen bg-background text-foreground">
      <NewProjectScreen
        onCancel={() =>
          void navigate({
            to: "/projects",
            search: { project: undefined, q: "" },
          })
        }
        onCreate={async (project) => {
          await createProject(project);
          await navigate({
            to: "/projects",
            search: { project: project.id, q: "" },
          });
        }}
      />
    </div>
  );
}
