import { ArrowUp, ImagePlus, LoaderCircle, Target, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import type { InspectorEditProps, InspectorReference } from "./inspector.types";
import { useInspectorEdit } from "./use-inspector-edit";

export function InspectorEdit(props: InspectorEditProps) {
  const {
    selectedNodes,
    selectedFrames,
    prompt,
    animations,
    onPromptChange,
    onSubmit,
    onClearSelection,
    isSubmitting = false,
    submitError,
  } = props;
  const controller = useInspectorEdit({
    selectedNodes,
    selectedFrames,
    prompt,
    animations,
    onSubmit,
    isSubmitting,
  });
  const { getInputProps, getRootProps, isDragActive, open } =
    controller.dropzone;

  return (
    <form
      className="overflow-hidden rounded-xl border bg-background shadow-sm"
      onSubmit={controller.handleSubmit}
    >
      <div
        {...getRootProps()}
        className={`min-h-56 p-3 transition-colors ${isDragActive ? "bg-primary/5" : ""}`}
      >
        <input {...getInputProps()} />
        <div className="flex items-center gap-2">
          <div className="grid size-7 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground">
            <Target className="size-3.5" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
              Target
            </p>
            <p className="truncate text-xs font-semibold">
              {controller.target.label}
            </p>
            <p className="truncate text-[11px] text-muted-foreground">
              {controller.target.detail}
            </p>
          </div>
          {controller.canClearSelection ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label="Clear selected target"
                    onClick={onClearSelection}
                  />
                }
              >
                <X />
              </TooltipTrigger>
              <TooltipContent>Use the entire asset</TooltipContent>
            </Tooltip>
          ) : null}
        </div>

        <Textarea
          aria-label="Edit prompt"
          className="mt-4 min-h-28 resize-none border-0 bg-transparent px-0 py-2 text-sm leading-6 shadow-none focus-visible:border-0 focus-visible:ring-0"
          placeholder="What would you like to change?"
          value={prompt}
          onChange={(event) => onPromptChange(event.target.value)}
          onKeyDown={controller.handlePromptKeyDown}
        />

        {controller.reference ? (
          <ReferencePreview
            reference={controller.reference}
            onClear={controller.clearReference}
          />
        ) : null}
        {isDragActive ? (
          <p className="mt-2 text-xs font-medium text-primary">
            Drop image to attach
          </p>
        ) : null}
      </div>

      {controller.referenceError || submitError ? (
        <p className="border-t px-3 py-2 text-xs text-destructive" role="alert">
          {controller.referenceError ?? submitError}
        </p>
      ) : null}

      <div className="flex items-center justify-between gap-2 border-t bg-muted/20 px-3 py-2">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label="Attach image"
                onClick={open}
              />
            }
          >
            <ImagePlus />
          </TooltipTrigger>
          <TooltipContent>Attach image</TooltipContent>
        </Tooltip>
        <div className="flex items-center gap-2">
          {controller.isReadingReference ? (
            <span className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <LoaderCircle className="size-3 animate-spin" />
              Reading image
            </span>
          ) : null}
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="submit"
                  size="icon"
                  aria-label="Send prompt"
                  title="Send prompt"
                  disabled={!controller.canSubmit}
                />
              }
            >
              {isSubmitting ? (
                <LoaderCircle className="animate-spin" />
              ) : (
                <ArrowUp />
              )}
            </TooltipTrigger>
            <TooltipContent>Send prompt</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </form>
  );
}

function ReferencePreview({
  reference,
  onClear,
}: {
  reference: InspectorReference;
  onClear: () => void;
}) {
  return (
    <div className="mt-3 flex items-center gap-2 rounded-lg border bg-muted/30 p-1.5">
      <img
        src={reference.dataUrl}
        alt={reference.fileName}
        className="size-10 rounded-md border object-cover"
      />
      <p className="min-w-0 flex-1 truncate text-xs font-medium">
        {reference.fileName}
      </p>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label="Remove attached image"
              onClick={onClear}
            />
          }
        >
          <X />
        </TooltipTrigger>
        <TooltipContent>Remove image</TooltipContent>
      </Tooltip>
    </div>
  );
}
