import { ArrowLeft, ArrowRight, FilePlus2 } from "lucide-react";

import type { NewProjectController } from "./use-new-project-controller";
import { ProjectStartCard } from "./project-start-card";

export function BlankProjectFlow({
  active,
  project,
}: {
  active: boolean;
  project: NewProjectController;
}) {
  if (!active) {
    return (
      <ProjectStartCard
        title="Blank project"
        description="Open a flexible workspace. Add context if useful, or create an asset immediately."
        icon={<FilePlus2 size={20} />}
        onSelect={project.start.chooseBlank}
      />
    );
  }

  const { form } = project;
  const newProjectForm = form.instance;

  return (
    <form
      className="mx-auto grid w-full max-w-2xl gap-6"
      onSubmit={(event) => {
        event.preventDefault();
        void newProjectForm.handleSubmit();
      }}
    >
      <newProjectForm.Field name="name">
        {(field) => (
          <label className="grid gap-2 text-sm font-semibold">
            Project name
            <input
              autoFocus
              value={field.state.value}
              onChange={(event) => field.handleChange(event.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
              placeholder="e.g. Moonlit Orchard"
            />
          </label>
        )}
      </newProjectForm.Field>
      <div className="mt-2 flex justify-between border-t pt-6">
        <button
          type="button"
          className="inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
          onClick={form.previous}
        >
          <ArrowLeft size={16} /> Previous
        </button>
        <button
          className="inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
          type="submit"
        >
          Submit
          <ArrowRight size={16} />
        </button>
      </div>
    </form>
  );
}
