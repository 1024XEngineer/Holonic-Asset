import { Pencil } from "lucide-react";
import { useForm, useStore } from "@tanstack/react-form";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { Textarea } from "@/components/ui/textarea";
import { DropdownField } from "@/components/ui/custom/dropdown-field";
import {
  applyProjectSettings,
  createProjectSettingsDraft,
  editableProjectContextOptions,
  projectContextOptions,
  updateProjectSettingsDraft,
  type ProjectSettingsDraft,
} from "./types";
import type { ProjectSummary } from "@/model/project";

export function ProjectSettingsDialog({
  project,
  onSave,
  iconOnly = false,
}: {
  project: ProjectSummary;
  onSave: (project: ProjectSummary) => void;
  iconOnly?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const form = useForm({
    defaultValues: { draft: createProjectSettingsDraft(project) },
    onSubmit: ({ value }) => {
      const updatedProject = applyProjectSettings(project, value.draft);
      if (!updatedProject) return;
      onSave(updatedProject);
      setOpen(false);
    },
  });
  const values = useStore(form.store, (state) => state.values.draft);

  useEffect(() => {
    form.reset({ draft: createProjectSettingsDraft(project) });
  }, [project]);

  function update<K extends keyof ProjectSettingsDraft>(
    key: K,
    value: ProjectSettingsDraft[K],
  ) {
    form.setFieldValue(
      "draft",
      updateProjectSettingsDraft(form.state.values.draft, key, value),
    );
  }

  function uploadDirection(file: File | undefined) {
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () =>
      update("visualDirection", String(reader.result ?? ""));
    reader.readAsDataURL(file);
  }

  function changeOpen(nextOpen: boolean) {
    if (nextOpen) {
      form.reset({ draft: createProjectSettingsDraft(project) });
    }
    setOpen(nextOpen);
  }

  return (
    <Dialog open={open} onOpenChange={changeOpen}>
      <DialogTrigger
        render={
          <Button
            variant="ghost"
            size={iconOnly ? "icon-sm" : "sm"}
            className={
              iconOnly
                ? "pointer-events-none opacity-0 text-muted-foreground transition-all group-hover:pointer-events-auto group-hover:opacity-100 hover:bg-foreground/10 hover:text-foreground focus-visible:pointer-events-auto focus-visible:opacity-100 focus-visible:text-foreground"
                : undefined
            }
            aria-label={`Edit ${project.name}`}
          />
        }
      >
        <Pencil data-icon={iconOnly ? undefined : "inline-start"} />
        {iconOnly ? (
          <span className="sr-only">Edit project</span>
        ) : (
          "Edit project"
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit project</DialogTitle>
          <DialogDescription>
            These defaults guide every asset generated inside this project.
          </DialogDescription>
        </DialogHeader>
        <form
          className="grid gap-5"
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="grid gap-2 text-sm font-medium sm:col-span-2">
              Project name
              <Input
                required
                value={values.name}
                onChange={(event) => update("name", event.target.value)}
              />
            </label>
            <DropdownField
              label="Game type"
              value={values.gameType}
              options={[...editableProjectContextOptions.gameTypes]}
              onChange={(value) => update("gameType", value)}
            />
            <DropdownField
              label="Visual style"
              value={values.visualStyle}
              options={[...editableProjectContextOptions.visualStyles]}
              onChange={(value) => update("visualStyle", value)}
            />
            {values.gameType === "Other" ? (
              <label
                className={`grid gap-2 text-sm font-medium ${
                  values.visualStyle === "Other" ? "" : "sm:col-span-2"
                }`}
              >
                Custom game type
                <Input
                  required
                  placeholder="Describe the game type"
                  value={values.customGameType}
                  onChange={(event) =>
                    update("customGameType", event.target.value)
                  }
                />
              </label>
            ) : null}
            {values.visualStyle === "Other" ? (
              <label
                className={`grid gap-2 text-sm font-medium ${
                  values.gameType === "Other" ? "" : "sm:col-span-2"
                }`}
              >
                Custom visual style
                <Input
                  required
                  placeholder="Describe the visual style"
                  value={values.customVisualStyle}
                  onChange={(event) =>
                    update("customVisualStyle", event.target.value)
                  }
                />
              </label>
            ) : null}
            <DropdownField
              label="Target platform"
              value={values.platform}
              options={[...projectContextOptions.platforms]}
              onChange={(value) => update("platform", value)}
            />
            <label className="grid gap-2 text-sm font-medium sm:col-span-2">
              Game description
              <Textarea
                className="min-h-28 resize-y"
                value={values.description}
                onChange={(event) => update("description", event.target.value)}
              />
            </label>
          </div>
          <div>
            <p className="text-sm font-medium">Visual direction</p>
            <ImageDropzone
              className="mt-2 h-28"
              previewUrl={values.visualDirection || undefined}
              onSelect={uploadDirection}
              onClear={() => update("visualDirection", "")}
            />
          </div>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>
              Cancel
            </DialogClose>
            <Button type="submit">Save changes</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
