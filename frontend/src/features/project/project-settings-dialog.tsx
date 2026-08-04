import { useForm } from "@tanstack/react-form";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
} from "./types";
import type { ProjectSummary } from "@/model/project";

export function ProjectSettingsDialog({
  project,
  onOpenChange,
  onSave,
}: {
  project: ProjectSummary;
  onOpenChange: (open: boolean) => void;
  onSave: (project: ProjectSummary) => void;
}) {
  const form = useForm({
    defaultValues: createProjectSettingsDraft(project),
    onSubmit: ({ value }) => {
      const updatedProject = applyProjectSettings(project, value);
      if (!updatedProject) return;
      onSave(updatedProject);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open onOpenChange={onOpenChange}>
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
            <form.Field name="name">
              {(field) => (
                <label className="grid gap-2 text-sm font-medium sm:col-span-2">
                  Project name
                  <Input
                    required
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </label>
              )}
            </form.Field>
            <form.Field name="gameType">
              {(field) => (
                <DropdownField
                  label="Game type"
                  value={field.state.value}
                  options={[...editableProjectContextOptions.gameTypes]}
                  onChange={(value) => {
                    field.handleChange(value);
                    if (value !== "Other")
                      form.setFieldValue("customGameType", "");
                  }}
                />
              )}
            </form.Field>
            <form.Field name="visualStyle">
              {(field) => (
                <DropdownField
                  label="Visual style"
                  value={field.state.value}
                  options={[...editableProjectContextOptions.visualStyles]}
                  onChange={(value) => {
                    field.handleChange(value);
                    if (value !== "Other")
                      form.setFieldValue("customVisualStyle", "");
                  }}
                />
              )}
            </form.Field>
            <form.Subscribe
              selector={(state) =>
                (state.values.gameType === "Other" ? 1 : 0) |
                (state.values.visualStyle === "Other" ? 2 : 0)
              }
            >
              {(customFieldMask) => {
                const showCustomGameType = (customFieldMask & 1) !== 0;
                const showCustomVisualStyle = (customFieldMask & 2) !== 0;

                return (
                  <>
                    {showCustomGameType ? (
                      <form.Field name="customGameType">
                        {(field) => (
                          <label
                            className={`grid gap-2 text-sm font-medium ${
                              showCustomVisualStyle ? "" : "sm:col-span-2"
                            }`}
                          >
                            Custom game type
                            <Input
                              required
                              placeholder="Describe the game type"
                              value={field.state.value}
                              onChange={(event) =>
                                field.handleChange(event.target.value)
                              }
                            />
                          </label>
                        )}
                      </form.Field>
                    ) : null}
                    {showCustomVisualStyle ? (
                      <form.Field name="customVisualStyle">
                        {(field) => (
                          <label
                            className={`grid gap-2 text-sm font-medium ${
                              showCustomGameType ? "" : "sm:col-span-2"
                            }`}
                          >
                            Custom visual style
                            <Input
                              required
                              placeholder="Describe the visual style"
                              value={field.state.value}
                              onChange={(event) =>
                                field.handleChange(event.target.value)
                              }
                            />
                          </label>
                        )}
                      </form.Field>
                    ) : null}
                  </>
                );
              }}
            </form.Subscribe>
            <form.Field name="platform">
              {(field) => (
                <DropdownField
                  label="Target platform"
                  value={field.state.value}
                  options={[...projectContextOptions.platforms]}
                  onChange={field.handleChange}
                />
              )}
            </form.Field>
            <form.Field name="description">
              {(field) => (
                <label className="grid gap-2 text-sm font-medium sm:col-span-2">
                  Game description
                  <Textarea
                    className="min-h-28 resize-y"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                </label>
              )}
            </form.Field>
          </div>
          <form.Field name="visualDirection">
            {(field) => (
              <div>
                <p className="text-sm font-medium">Visual direction</p>
                <ImageDropzone
                  className="mt-2 h-28"
                  previewUrl={field.state.value || undefined}
                  onSelect={(file) => {
                    const reader = new FileReader();
                    reader.onload = () =>
                      field.handleChange(String(reader.result ?? ""));
                    reader.readAsDataURL(file);
                  }}
                  onClear={() => field.handleChange("")}
                />
              </div>
            )}
          </form.Field>
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
