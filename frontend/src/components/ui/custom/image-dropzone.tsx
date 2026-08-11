/**
 * Custom Component: ImageDropzone
 * Image upload and preview dropzone component built with react-dropzone.
 */

import { ImagePlus, LoaderCircle, RefreshCw, Upload, X } from "lucide-react";
import { useDropzone } from "react-dropzone";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const imageAccept = {
  "image/jpeg": [".jpg", ".jpeg"],
  "image/png": [".png"],
  "image/webp": [".webp"],
};

export function ImageDropzone({
  className,
  fileName,
  error,
  label = "Upload a reference image",
  onClear,
  onPreview,
  onRegenerate,
  onSelect,
  isRegenerating = false,
  previewUrl,
}: {
  className?: string;
  fileName?: string;
  error?: string;
  label?: string;
  onClear?: () => void;
  onPreview?: () => void;
  onRegenerate?: () => void;
  onSelect: (file: File) => void;
  isRegenerating?: boolean;
  previewUrl?: string;
}) {
  const { fileRejections, getInputProps, getRootProps, isDragActive, open } =
    useDropzone({
      accept: imageAccept,
      maxFiles: 1,
      multiple: false,
      onDropAccepted: ([file]) => file && onSelect(file),
    });

  return (
    <div className="grid gap-2">
      <div
        {...getRootProps()}
        className={cn(
          "group relative flex min-h-28 cursor-pointer items-center justify-center overflow-visible rounded-lg border border-dashed bg-muted/30 text-sm text-muted-foreground transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
          isDragActive && "border-ring bg-muted",
          className,
        )}
      >
        <input {...getInputProps()} />
        {previewUrl ? (
          <div className="relative size-full overflow-hidden rounded-[inherit]">
            <img
              src={previewUrl}
              alt="Selected reference"
              className={cn(
                "size-full object-cover",
                onPreview && "cursor-zoom-in",
                isRegenerating && "scale-105 opacity-45 blur-md",
              )}
              onClick={
                onPreview
                  ? (event) => {
                      event.stopPropagation();
                      onPreview();
                    }
                  : undefined
              }
              onKeyDown={
                onPreview
                  ? (event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        event.stopPropagation();
                        onPreview();
                      }
                    }
                  : undefined
              }
              role={onPreview ? "button" : undefined}
              tabIndex={onPreview ? 0 : undefined}
              aria-label={onPreview ? "Preview reference image" : undefined}
            />
            {isRegenerating ? (
              <div className="pointer-events-none absolute inset-0 grid place-items-center bg-background/25">
                <LoaderCircle className="size-8 animate-spin text-foreground" />
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
        {previewUrl && onRegenerate && !isRegenerating ? (
          <div className="absolute top-2 right-2 flex gap-1">
            <div className="group/action relative">
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                className="bg-background/90"
                aria-label="Regenerate reference"
                onClick={(event) => {
                  event.stopPropagation();
                  onRegenerate();
                }}
              >
                <RefreshCw />
              </Button>
              <span className="pointer-events-none absolute bottom-full left-1/2 mb-2 -translate-x-1/2 whitespace-nowrap rounded-md bg-black px-2.5 py-1.5 text-xs text-white opacity-0 shadow-md transition-opacity group-hover/action:opacity-100">
                Regenerate reference
              </span>
            </div>
            <div className="group/action relative">
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                className="bg-background/90"
                aria-label="Upload reference image"
                onClick={(event) => {
                  event.stopPropagation();
                  open();
                }}
              >
                <Upload />
              </Button>
              <span className="pointer-events-none absolute bottom-full left-1/2 mb-2 -translate-x-1/2 whitespace-nowrap rounded-md bg-black px-2.5 py-1.5 text-xs text-white opacity-0 shadow-md transition-opacity group-hover/action:opacity-100">
                Upload reference image
              </span>
            </div>
          </div>
        ) : null}
        {onClear && (previewUrl || fileName) ? (
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            className="absolute top-2 left-2 bg-background/90"
            aria-label="Remove reference image"
            onClick={(event) => {
              event.stopPropagation();
              onClear();
            }}
          >
            <X />
          </Button>
        ) : null}
      </div>
      {fileRejections.length > 0 ? (
        <p className="text-xs text-destructive">
          Use a PNG, JPEG, or WebP image.
        </p>
      ) : error ? (
        <p className="text-xs text-destructive">{error}</p>
      ) : null}
    </div>
  );
}
