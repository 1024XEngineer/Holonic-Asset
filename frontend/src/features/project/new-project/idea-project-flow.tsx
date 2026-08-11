import { Lightbulb } from "lucide-react";

import type { NewProjectController } from "./use-new-project-controller";
import { GuidedProjectFlow } from "./guided-project-flow";
import { ProjectStartCard } from "./project-start-card";

export function IdeaProjectFlow({
  active,
  project,
}: {
  active: boolean;
  project: NewProjectController;
}) {
  if (active) return <GuidedProjectFlow project={project} />;

  return (
    <ProjectStartCard
      title="I have an idea"
      description="Describe the game, generate a reference, and refine it until it feels right."
      icon={<Lightbulb size={20} />}
      onSelect={project.start.chooseIdea}
    />
  );
}
