import { useEffect, useRef, useState, type ReactNode } from "react";
import { useForm } from "@tanstack/react-form";
import { Eye, LoaderCircle, RefreshCw, X } from "lucide-react";

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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { DropdownField } from "@/components/ui/custom/dropdown-field";
import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";
import { isPerspective, projectApi } from "@/model/project";
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
  const [imageError, setImageError] = useState<string>();
  const [isReferencePreviewOpen, setIsReferencePreviewOpen] = useState(false);
  const [isRegeneratingReference, setIsRegeneratingReference] = useState(false);
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
      <DialogContent
        className={`flex max-h-[calc(100dvh-2rem)] min-w-0 flex-col overflow-hidden transition-[filter] duration-150 sm:max-w-2xl ${isReferencePreviewOpen ? "blur-sm" : ""}`}
      >
        <DialogHeader>
          <DialogTitle>Edit project</DialogTitle>
          <DialogDescription>
            These defaults guide every asset generated inside this project.
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex min-h-0 min-w-0 flex-1 flex-col gap-5 overflow-hidden"
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        >
          <div className="grid min-h-0 min-w-0 flex-1 gap-5 overflow-x-hidden overflow-y-auto pr-1 sm:grid-cols-2">
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
            <form.Field name="perspective">
              {(field) => (
                <DropdownField
                  label="Perspective"
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
                          placeholder="Describe the game type"
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
                    label="Target platform"
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
            <form.Field name="reference">
              {(field) => (
                <div className="min-w-0 sm:col-span-2">
                  <p className="text-sm font-medium">Reference</p>
                  <div className="relative mt-2">
                    <ImageDropzone
                      className="aspect-[16/9] min-h-48 min-w-0 max-w-full"
                      value={field.state.value || undefined}
                      error={imageError}
                      onChange={(file) => {
                        imageReadController.current?.abort();
                        setImageError(undefined);
                        if (!file) {
                          field.handleChange("");
                          return;
                        }
                        const controller = new AbortController();
                        imageReadController.current = controller;
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
                    />
                    {field.state.value ? (
                      <ProjectReferenceActions
                        isRegenerating={isRegeneratingReference}
                        onClear={() => {
                          imageReadController.current?.abort();
                          setImageError(undefined);
                          field.handleChange("");
                        }}
                        onPreview={() => setIsReferencePreviewOpen(true)}
                        onRegenerate={() => {
                          const updatedProject = applyProjectSettings(
                            project,
                            form.state.values,
                          );
                          if (!updatedProject) {
                            setImageError(
                              "Complete the project details first.",
                            );
                            return;
                          }
                          setIsRegeneratingReference(true);
                          setImageError(undefined);
                          void projectApi
                            .regenerateReference(updatedProject)
                            .then((reference) => field.handleChange(reference))
                            .catch(() =>
                              setImageError(
                                "We couldn't regenerate that reference. Try again.",
                              ),
                            )
                            .finally(() => setIsRegeneratingReference(false));
                        }}
                      />
                    ) : null}
                  </div>
                  <Dialog
                    open={isReferencePreviewOpen}
                    onOpenChange={setIsReferencePreviewOpen}
                  >
                    <DialogContent
                      showCloseButton={false}
                      className="!inset-0 !flex !h-dvh !w-screen !max-w-none !translate-x-0 !translate-y-0 items-center justify-center bg-transparent p-0 ring-0"
                      onClick={(event) => {
                        if (event.target === event.currentTarget)
                          setIsReferencePreviewOpen(false);
                      }}
                    >
                      <DialogTitle className="sr-only">
                        Project reference
                      </DialogTitle>
                      {field.state.value ? (
                        <img
                          src={field.state.value}
                          alt="Project reference"
                          className="max-h-[80vh] max-w-[calc(100vw-2rem)] object-contain"
                        />
                      ) : null}
                    </DialogContent>
                  </Dialog>
                </div>
              )}
            </form.Field>
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

function ProjectReferenceActions({
  isRegenerating,
  onClear,
  onPreview,
  onRegenerate,
}: {
  isRegenerating: boolean;
  onClear: () => void;
  onPreview: () => void;
  onRegenerate: () => void;
}) {
  return (
    <div className="absolute right-2 top-2 z-10 flex gap-1">
      <ReferenceAction
        label="Preview reference image"
        disabled={isRegenerating}
        onClick={onPreview}
      >
        <Eye />
      </ReferenceAction>
      <ReferenceAction
        label="Regenerate reference image"
        disabled={isRegenerating}
        onClick={onRegenerate}
      >
        {isRegenerating ? (
          <LoaderCircle className="animate-spin" />
        ) : (
          <RefreshCw />
        )}
      </ReferenceAction>
      <ReferenceAction
        label="Remove reference image"
        disabled={isRegenerating}
        onClick={onClear}
      >
        <X />
      </ReferenceAction>
    </div>
  );
}

function ReferenceAction({
  children,
  disabled,
  label,
  onClick,
}: {
  children: ReactNode;
  disabled: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            className="bg-background/90 shadow-xs backdrop-blur-sm"
            aria-label={label}
            disabled={disabled}
            onClick={(event) => {
              event.stopPropagation();
              onClick();
            }}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
