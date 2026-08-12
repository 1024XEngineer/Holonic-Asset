import { ArrowLeft, ArrowRight, FilePlus2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { NewProjectController } from "./use-new-project-controller";
import { ProjectStartCard } from "./project-start-card";

export function BlankProjectFlow({
  active,
  project,
}: {
  active: boolean;
  project: NewProjectController;
}) {
  const { t } = useTranslation("projects");
  if (!active) {
    return (
      <ProjectStartCard
        title={t("blank")}
        description={t("blankDescription")}
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
            {t("projectName")}
            <input
              autoFocus
              value={field.state.value}
              onChange={(event) => field.handleChange(event.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
              placeholder={t("projectNamePlaceholder")}
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
          <ArrowLeft size={16} /> {t("previous")}
        </button>
        <button
          className="inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
          type="submit"
        >
          {t("submit")}
          <ArrowRight size={16} />
        </button>
      </div>
    </form>
  );
}
