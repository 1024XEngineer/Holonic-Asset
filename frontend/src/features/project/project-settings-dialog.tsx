import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
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
import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";
import { isPerspective } from "@/model/project";
import {
  applyProjectSettings,
  createProjectSettingsDraft,
  editableProjectContextOptions,
  projectContextOptions,
} from "./lib/project-context";
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
  const { t } = useTranslation("projects");
  const [imageError, setImageError] = useState<string>();
  const imageReadController = useRef<AbortController | null>(null);
  const form = useForm({
    defaultValues: createProjectSettingsDraft(project),
    onSubmit: ({ value }) => {
      const updatedProject = applyProjectSettings(project, value);
      if (!updatedProject) return;
      onSave(updatedProject);
      onOpenChange(false);
    },
  });

  useEffect(() => () => imageReadController.current?.abort(), []);

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("edit")}</DialogTitle>
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
                  label={t("gameType")}
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
            <form.Field name="perspective">
              {(field) => (
                <DropdownField
                  label={t("perspective")}
                  value={field.state.value}
                  options={projectContextOptions.perspectives}
                  onChange={(value) => {
                    if (isPerspective(value)) field.handleChange(value);
                  }}
                />
              )}
            </form.Field>
            <form.Subscribe
              selector={(state) => state.values.gameType === "Other"}
            >
              {(showCustomGameType) =>
                showCustomGameType ? (
                  <form.Field name="customGameType">
                    {(field) => (
                      <label className="grid gap-2 text-sm font-medium sm:col-span-2">
                        Custom game type
                        <Input
                          required
                          placeholder={t("describeGameType")}
                          value={field.state.value}
                          onChange={(event) =>
                            field.handleChange(event.target.value)
                          }
                        />
                      </label>
                    )}
                  </form.Field>
                ) : null
              }
            </form.Subscribe>
            <div className="sm:col-span-2">
              <form.Field name="platform">
                {(field) => (
                  <DropdownField
                    label={t("platform")}
                    value={field.state.value}
                    options={[...projectContextOptions.platforms]}
                    onChange={field.handleChange}
                  />
                )}
              </form.Field>
            </div>
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
                <p className="text-sm font-medium">{t("reference")}</p>
                <ImageDropzone
                  className="mt-2 h-28"
                  previewUrl={field.state.value || undefined}
                  error={imageError}
                  onSelect={(file) => {
                    imageReadController.current?.abort();
                    const controller = new AbortController();
                    imageReadController.current = controller;
                    setImageError(undefined);
                    void readFileAsDataUrl(file, controller.signal).then(
                      (dataUrl) => {
                        if (controller.signal.aborted) return;
                        field.handleChange(dataUrl);
                      },
                      () => {
                        if (controller.signal.aborted) return;
                        setImageError(
                          "We couldn't read that image. Try another file.",
                        );
                      },
                    );
                  }}
                  onClear={() => {
                    imageReadController.current?.abort();
                    setImageError(undefined);
                    field.handleChange("");
                  }}
                />
              </div>
            )}
          </form.Field>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>
              Cancel
            </DialogClose>
            <Button type="submit">{t("save")}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
