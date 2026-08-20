import { ArrowUp, ImagePlus, LoaderCircle, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import type {
  InspectorReference,
  SpriteInspectorContentProps,
} from "./inspector.types";
import { useInspectorEdit } from "./use-inspector-edit";

export function SpriteInspectorContent(props: SpriteInspectorContentProps) {
  const { t } = useTranslation("editor");
  const {
    selectedNodes,
    selectedFrames,
    prompt,
    animations,
    prototype,
    onPromptChange,
    onSubmit,
    onClearSelection,
    isSubmitting = false,
  } = props;
  const controller = useInspectorEdit({
    selectedNodes,
    selectedFrames,
    prompt,
    animations,
    prototype,
    onPromptChange,
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
        {controller.target ? (
          <div className="flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/5 px-2.5 py-2 text-xs shadow-sm">
            <TargetThumbnail target={controller.target} />
            <div className="min-w-0 flex-1">
              <p className="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
                {t("target")}
              </p>
              <p className="truncate font-semibold">
                {controller.target.label}
              </p>
            </div>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("clearSelectedTarget")}
                    onClick={onClearSelection}
                  />
                }
              >
                <X />
              </TooltipTrigger>
              <TooltipContent>{t("useEntireAsset")}</TooltipContent>
            </Tooltip>
          </div>
        ) : null}

        <Textarea
          aria-label={t("editPrompt")}
          className={`${controller.target ? "mt-3" : ""} min-h-28 resize-none border-0 bg-transparent px-0 py-2 text-sm leading-6 shadow-none focus-visible:border-0 focus-visible:ring-0`}
          placeholder={t("promptPlaceholder")}
          value={prompt}
          onChange={(event) => controller.changePrompt(event.target.value)}
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
            {t("dropImage")}
          </p>
        ) : null}
      </div>

      {controller.referenceError || controller.submitError ? (
        <p className="border-t px-3 py-2 text-xs text-destructive" role="alert">
          {controller.referenceError ?? controller.submitError}
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
                aria-label={t("attachImage")}
                onClick={open}
              />
            }
          >
            <ImagePlus />
          </TooltipTrigger>
          <TooltipContent>{t("attachImage")}</TooltipContent>
        </Tooltip>
        <div className="flex items-center gap-2">
          {controller.isReadingReference ? (
            <span className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <LoaderCircle className="size-3 animate-spin" />
              {t("readingImage")}
            </span>
          ) : null}
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="submit"
                  size="icon"
                  aria-label={t("sendPrompt")}
                  title={t("sendPrompt")}
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
            <TooltipContent>{t("sendPrompt")}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </form>
  );
}

function TargetThumbnail({
  target,
}: {
  target: NonNullable<ReturnType<typeof useInspectorEdit>>["target"];
}) {
  if (!target) return null;
  const { thumbnail } = target;
  const imageWidth = thumbnail.columns * 100;
  const imageHeight = thumbnail.rows * 100;

  return (
    <div className="size-7 shrink-0 overflow-hidden rounded-md border bg-muted">
      <img
        src={thumbnail.imageUrl}
        alt=""
        className="block max-w-none object-none"
        style={{
          width: `${imageWidth}%`,
          height: `${imageHeight}%`,
          transform: `translate(-${(thumbnail.column * 100) / thumbnail.columns}%, -${(thumbnail.row * 100) / thumbnail.rows}%)`,
        }}
      />
    </div>
  );
}

function ReferencePreview({
  reference,
  onClear,
}: {
  reference: InspectorReference;
  onClear: () => void;
}) {
  const { t } = useTranslation("editor");
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
              aria-label={t("removeImage")}
              onClick={onClear}
            />
          }
        >
          <X />
        </TooltipTrigger>
        <TooltipContent>{t("removeImage")}</TooltipContent>
      </Tooltip>
    </div>
  );
}
