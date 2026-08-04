import { ArrowLeft } from "lucide-react";

import { BlankProjectFlow } from "./blank-project-flow";
import { ExistingGameFlow } from "./existing-game-flow";
import { IdeaProjectFlow } from "./idea-project-flow";
import type { NewProjectController } from "./use-new-project-controller";

export interface NewProjectWorkspaceProps {
  project: NewProjectController;
}

export function NewProjectWorkspace({ project }: NewProjectWorkspaceProps) {
  const { backToLibrary, form } = project;
  const { selectedStart } = form;
  const isBlank = selectedStart === "blank";

  return (
    <main className="min-h-screen bg-background">
      <div className="mx-auto w-full max-w-6xl px-4 py-8 pb-20 sm:px-6">
        <button
          type="button"
          onClick={selectedStart ? form.returnToStart : backToLibrary}
          className="mb-8 inline-flex items-center gap-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" />
          Back
        </button>

        <div className="mx-auto max-w-2xl">
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">
              {selectedStart
                ? isBlank
                  ? "Start with as little as you like"
                  : "Tell us about your game"
                : "Where would you like to start?"}
            </h1>
            <p className="mt-2 text-muted-foreground">
              {selectedStart
                ? isBlank
                  ? "Give your project a name. You can add details whenever you are ready."
                  : "Add a few details to help shape your project."
                : "Pick the option that best matches where you are today."}
            </p>
          </div>

          {!selectedStart ? (
            <div className="grid gap-4 sm:grid-cols-3">
              <ExistingGameFlow active={false} project={project} />
              <IdeaProjectFlow active={false} project={project} />
              <BlankProjectFlow active={false} project={project} />
            </div>
          ) : selectedStart === "existing" ? (
            <ExistingGameFlow active project={project} />
          ) : selectedStart === "idea" ? (
            <IdeaProjectFlow active project={project} />
          ) : (
            <BlankProjectFlow active project={project} />
          )}
        </div>
      </div>
    </main>
  );
}
