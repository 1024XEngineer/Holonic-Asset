import { Lightbulb } from "lucide-react";
import { useTranslation } from "react-i18next";

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
  const { t } = useTranslation("workspace");
  if (active) return <GuidedProjectFlow project={project} />;

  return (
    <ProjectStartCard
      title={t("project.idea")}
      description={t("project.ideaDescription")}
      icon={<Lightbulb size={20} />}
      onSelect={project.start.chooseIdea}
    />
  );
}
