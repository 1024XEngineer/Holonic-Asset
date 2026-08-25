import { ArrowLeft } from "lucide-react";
import { useTranslation } from "react-i18next";

import { BlankProjectFlow } from "./blank-project-flow";
import { ExistingGameFlow } from "./existing-game-flow";
import { IdeaProjectFlow } from "./idea-project-flow";
import type { NewProjectController } from "./use-new-project-controller";

export interface NewProjectWorkspaceProps {
  project: NewProjectController;
}

export function NewProjectWorkspace({ project }: NewProjectWorkspaceProps) {
  const { t } = useTranslation(["projects", "common"]);
  const { backToLibrary, form } = project;
  const { selectedStart } = form;
  const isBlank = selectedStart === "blank";
  const hasCenteredForm = isBlank || selectedStart === "idea";
  let title = t("startTitle");
  let description = t("startPrompt");
  if (selectedStart === "blank") {
    title = t("startBlankTitle");
    description = t("blankPrompt");
  } else if (selectedStart) {
    title = t("tellGameTitle");
    description = t("detailsPrompt");
  }

  let formContent: React.ReactNode;
  if (!selectedStart) {
    formContent = (
      <div className="flex flex-1 items-center">
        <div className="mx-auto grid w-full max-w-6xl gap-x-4 gap-y-8 sm:grid-cols-3">
          <ExistingGameFlow active={false} project={project} />
          <IdeaProjectFlow active={false} project={project} />
          <BlankProjectFlow active={false} project={project} />
        </div>
      </div>
    );
  } else if (selectedStart === "existing") {
    formContent = <ExistingGameFlow active project={project} />;
  } else if (selectedStart === "idea") {
    formContent = (
      <div className="flex flex-1 items-center">
        <IdeaProjectFlow active project={project} />
      </div>
    );
  } else {
    formContent = (
      <div className="flex flex-1 items-center">
        <BlankProjectFlow active project={project} />
      </div>
    );
  }

  return (
    <main className="relative min-h-screen bg-background">
      <button
        type="button"
        onClick={selectedStart ? form.returnToStart : backToLibrary}
        className="absolute left-4 top-4 inline-flex items-center gap-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground sm:left-6 sm:top-6"
      >
        <ArrowLeft className="size-4" />
        {t("common:actions.back")}
      </button>
      <div
        className={`mx-auto w-full max-w-6xl px-4 py-8 pb-20 sm:px-6 ${
          !selectedStart || hasCenteredForm ? "flex min-h-screen flex-col" : ""
        }`}
      >
        <div className="mx-auto max-w-2xl text-center">
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">
              {title}
            </h1>
            <p className="mt-2 text-muted-foreground">{description}</p>
          </div>
        </div>

        {formContent}
      </div>
    </main>
  );
}
