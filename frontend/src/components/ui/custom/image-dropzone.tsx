/**
 * Custom Component: ImageDropzone
 * Image upload and preview dropzone component built with react-dropzone.
 */

import { ImagePlus, LoaderCircle, RefreshCw, Upload, X } from "lucide-react";
import { useId, type ReactNode } from "react";
import { useDropzone, type Accept, type FileRejection } from "react-dropzone";

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

const IMAGE_ACCEPT = {
  "image/jpeg": [".jpg", ".jpeg"],
  "image/png": [".png"],
  "image/webp": [".webp"],
} satisfies Accept;

const INVALID_IMAGE_MESSAGE = "Use a PNG, JPEG, or WebP image.";

export type ImageDropzoneValue = File | string;

export interface ImageDropzoneActions {
  loading?: boolean;
  preview?: () => void;
  regenerate?: () => void;
}

export interface ImageDropzoneProps {
  actions?: ImageDropzoneActions;
  className?: string;
  error?: string;
  label?: string;
  onChange: (value: File | undefined) => void;
  value?: ImageDropzoneValue;
}

export function ImageDropzone({
  actions,
  className,
  error,
  label = "Upload a reference image",
  onChange,
  value,
}: ImageDropzoneProps) {
  const {
    loading = false,
    preview: onPreview,
    regenerate: onRegenerate,
  } = actions ?? {};
  const errorId = useId();
  const previewUrl = typeof value === "string" ? value : undefined;
  const fileName = value instanceof File ? value.name : undefined;
  const hasSelection = Boolean(value);
  const { fileRejections, getInputProps, getRootProps, isDragActive, open } =
    useDropzone({
      accept: IMAGE_ACCEPT,
      disabled: loading,
      maxFiles: 1,
      multiple: false,
      onDropAccepted: ([file]) => file && onChange(file),
      noClick: hasSelection,
      noKeyboard: hasSelection,
    });
  const message = getRejectionMessage(fileRejections) ?? error;

  return (
    <div className="grid gap-2">
      <div
        {...getRootProps({
          "aria-describedby": message ? errorId : undefined,
          "aria-invalid": message ? true : undefined,
          "aria-label": hasSelection ? "Reference image" : label,
          role: hasSelection ? "group" : "button",
          tabIndex: hasSelection ? -1 : 0,
        })}
        className={cn(
          "group relative flex min-h-28 items-center justify-center overflow-visible rounded-lg border border-dashed bg-muted/30 text-sm text-muted-foreground transition-colors aria-invalid:border-destructive",
          hasSelection ? "cursor-default" : "cursor-pointer hover:bg-muted/60",
          isDragActive && "border-ring bg-muted",
          !hasSelection &&
            "focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
          className,
        )}
      >
        <input
          {...getInputProps({
            "aria-describedby": message ? errorId : undefined,
            "aria-invalid": message ? true : undefined,
          })}
        />
        {previewUrl ? (
          <div className="relative size-full overflow-hidden rounded-[inherit]">
            {onPreview ? (
              <button
                type="button"
                className="size-full cursor-zoom-in outline-none focus-visible:ring-3 focus-visible:ring-inset focus-visible:ring-ring/50 disabled:cursor-wait"
                aria-label="Preview reference image"
                disabled={loading}
                onClick={onPreview}
              >
                <PreviewImage
                  previewUrl={previewUrl}
                  isRegenerating={loading}
                />
              </button>
            ) : (
              <PreviewImage previewUrl={previewUrl} isRegenerating={loading} />
            )}
            {loading ? (
              <div
                className="pointer-events-none absolute inset-0 grid place-items-center bg-background/25"
                role="status"
              >
                <LoaderCircle
                  className="size-8 animate-spin text-foreground"
                  aria-hidden="true"
                />
                <span className="sr-only">Regenerating reference image</span>
              </div>
            ) : null}
          </div>
        ) : fileName ? (
          <span className="max-w-full truncate px-12">{fileName}</span>
        ) : (
          <span className="flex items-center gap-2">
            <ImagePlus className="size-4" />
            {isDragActive ? "Drop image to attach" : label}
          </span>
        )}
        {hasSelection ? (
          <div className="absolute right-2 top-2 z-10 flex gap-1">
            {previewUrl && onRegenerate ? (
              <DropzoneAction
                label="Regenerate reference"
                disabled={loading}
                onClick={onRegenerate}
              >
                <RefreshCw />
              </DropzoneAction>
            ) : null}
            <DropzoneAction
              label="Upload reference image"
              disabled={loading}
              onClick={open}
            >
              <Upload />
            </DropzoneAction>
          </div>
        ) : null}
        {hasSelection ? (
          <DropzoneAction
            className="absolute left-2 top-2 z-10"
            label="Remove reference image"
            disabled={loading}
            onClick={() => onChange(undefined)}
          >
            <X />
          </DropzoneAction>
        ) : null}
      </div>
      {message ? (
        <p id={errorId} className="text-xs text-destructive" role="alert">
          {message}
        </p>
      ) : null}
    </div>
  );
}

function PreviewImage({
  isRegenerating,
  previewUrl,
}: {
  isRegenerating: boolean;
  previewUrl: string;
}) {
  return (
    <img
      src={previewUrl}
      alt="Selected reference"
      className={cn(
        "size-full object-cover transition-[filter,opacity,transform]",
        isRegenerating && "scale-105 opacity-45 blur-md",
      )}
    />
  );
}

function DropzoneAction({
  children,
  className,
  disabled,
  label,
  onClick,
}: {
  children: ReactNode;
  className?: string;
  disabled: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <div className={className}>
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
    </div>
  );
}

function getRejectionMessage(fileRejections: readonly FileRejection[]) {
  if (fileRejections.length === 0) return undefined;

  const hasTooManyFiles = fileRejections.some(({ errors }) =>
    errors.some(({ code }) => code === "too-many-files"),
  );

  return hasTooManyFiles
    ? "Upload one image at a time."
    : INVALID_IMAGE_MESSAGE;
}
