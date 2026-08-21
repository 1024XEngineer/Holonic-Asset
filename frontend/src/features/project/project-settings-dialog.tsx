import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useForm } from "@tanstack/react-form";
import { ImageIcon, LoaderCircle, RefreshCw, Upload } from "lucide-react";

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
import { Textarea } from "@/components/ui/textarea";
import { DropdownField } from "@/components/ui/custom/dropdown-field";
import { isPerspective, projectApi } from "@/model/project";
import { uploadFile } from "@/model/upload";
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
  const { t } = useTranslation(["projects", "common"]);
  const [imageError, setImageError] = useState<string>();
  const [isGeneratingReference, setIsGeneratingReference] = useState(false);
  const [isUploadingReference, setIsUploadingReference] = useState(false);
  const [referencePreview, setReferencePreview] = useState(project.reference);
  const [previewImage, setPreviewImage] = useState<string>();
  const imageInputRef = useRef<HTMLInputElement>(null);
  const imageUploadController = useRef<AbortController | null>(null);
  const form = useForm({
    defaultValues: createProjectSettingsDraft(project),
    onSubmit: ({ value }) => {
      const updatedProject = applyProjectSettings(project, value);
      if (!updatedProject) return;
      onSave(updatedProject);
      onOpenChange(false);
    },
  });

  useEffect(() => () => imageUploadController.current?.abort(), []);

  return (
    <>
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
                      onChange={(event) =>
                        field.handleChange(event.target.value)
                      }
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
                      onChange={(event) =>
                        field.handleChange(event.target.value)
                      }
                    />
                  </label>
                )}
              </form.Field>
            </div>
            <form.Field name="reference">
              {(field) => (
                <div className="relative grid gap-2">
                  <p className="text-sm font-medium">{t("reference")}</p>
                  {referencePreview ? (
                    <ReferenceImage
                      src={referencePreview}
                      alt={t("referenceImageAlt")}
                      onPreview={() => setPreviewImage(referencePreview)}
                    />
                  ) : (
                    <div className="flex h-28 w-full items-center justify-center rounded-lg border border-dashed bg-muted/30 text-sm text-muted-foreground">
                      {t("uploadReference")}
                    </div>
                  )}
                  <div className="pointer-events-none absolute inset-x-0 mt-[1.75rem] flex h-28 justify-end p-2">
                    <div className="pointer-events-auto flex items-start gap-1.5">
                      <input
                        ref={imageInputRef}
                        type="file"
                        accept="image/jpeg,image/png,image/webp"
                        className="sr-only"
                        onChange={(event) => {
                          const file = event.target.files?.[0];
                          event.target.value = "";
                          if (!file) return;
                          if (
                            !["image/jpeg", "image/png", "image/webp"].includes(
                              file.type,
                            )
                          ) {
                            setImageError("Use a PNG, JPEG, or WebP image.");
                            return;
                          }

                          imageUploadController.current?.abort();
                          setImageError(undefined);
                          const controller = new AbortController();
                          imageUploadController.current = controller;
                          setIsUploadingReference(true);
                          void (async () => {
                            try {
                              const target = await uploadFile(
                                file,
                                controller.signal,
                              );
                              if (controller.signal.aborted) return;
                              field.handleChange(target.objectURL);
                              setReferencePreview(target.objectURL);
                            } catch {
                              if (controller.signal.aborted) return;
                              setImageError(
                                "We couldn't upload that image. Try again.",
                              );
                            } finally {
                              if (
                                imageUploadController.current === controller
                              ) {
                                imageUploadController.current = null;
                                setIsUploadingReference(false);
                              }
                            }
                          })();
                        }}
                      />
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        className="bg-background/90 shadow-sm backdrop-blur-sm"
                        disabled={isUploadingReference || isGeneratingReference}
                        onClick={() => imageInputRef.current?.click()}
                      >
                        {isUploadingReference ? (
                          <LoaderCircle className="animate-spin" />
                        ) : (
                          <Upload data-icon="inline-start" />
                        )}
                        {t("upload")}
                      </Button>
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        className="bg-background/90 shadow-sm backdrop-blur-sm"
                        disabled={isGeneratingReference || isUploadingReference}
                        onClick={async () => {
                          const updatedProject = applyProjectSettings(
                            project,
                            form.state.values,
                          );
                          if (!updatedProject) return;

                          setIsGeneratingReference(true);
                          setImageError(undefined);
                          try {
                            const reference =
                              await projectApi.regenerateReference(
                                updatedProject,
                              );
                            field.handleChange(reference);
                            setReferencePreview(reference);
                          } catch {
                            setImageError(
                              "We couldn't generate that reference. Try again.",
                            );
                          } finally {
                            setIsGeneratingReference(false);
                          }
                        }}
                      >
                        {isGeneratingReference ? (
                          <LoaderCircle className="animate-spin" />
                        ) : (
                          <RefreshCw data-icon="inline-start" />
                        )}
                        {t("regeneratePreview")}
                      </Button>
                    </div>
                  </div>
                  {imageError ? (
                    <p className="text-xs text-destructive" role="alert">
                      {imageError}
                    </p>
                  ) : null}
                </div>
              )}
            </form.Field>
            <DialogFooter>
              <DialogClose render={<Button type="button" variant="outline" />}>
                {t("common:actions.cancel")}
              </DialogClose>
              <Button
                type="submit"
                disabled={isUploadingReference || isGeneratingReference}
              >
                {t("save")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(previewImage)}
        onOpenChange={(open) => {
          if (!open) setPreviewImage(undefined);
        }}
      >
        <DialogContent
          showCloseButton={false}
          className="flex h-[92vh] w-[96vw] max-w-none items-center justify-center rounded-none border-none bg-transparent p-0 shadow-none ring-0"
        >
          <DialogTitle className="sr-only">{t("previewReference")}</DialogTitle>
          {previewImage ? (
            <img
              src={previewImage}
              alt={t("referenceImageAlt")}
              className="max-h-[92vh] max-w-[96vw] object-contain"
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  );
}

function ReferenceImage({
  alt,
  onPreview,
  src,
}: {
  alt: string;
  onPreview: () => void;
  src: string;
}) {
  return (
    <button
      type="button"
      className="group relative flex h-28 w-full items-center justify-center overflow-hidden rounded-lg border bg-muted/30 outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
      aria-label={alt}
      onClick={onPreview}
    >
      <img
        src={src}
        alt={alt}
        className="size-full object-cover transition-transform duration-200 group-hover:scale-[1.02]"
      />
      <span className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/0 text-white opacity-0 transition-all group-hover:bg-black/35 group-hover:opacity-100">
        <ImageIcon className="size-5" />
        <span className="sr-only">{alt}</span>
      </span>
    </button>
  );
}
