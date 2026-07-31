/**
 * Custom Component: ImageDropzone
 * Image upload and preview dropzone component built with react-dropzone.
 */

import { ImagePlus, X } from "lucide-react";
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
  label = "Upload a reference image",
  onClear,
  onSelect,
  previewUrl,
}: {
  className?: string;
  fileName?: string;
  label?: string;
  onClear?: () => void;
  onSelect: (file: File) => void;
  previewUrl?: string;
}) {
  const { fileRejections, getInputProps, getRootProps, isDragActive } =
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
          "group relative flex min-h-28 cursor-pointer items-center justify-center overflow-hidden rounded-lg border border-dashed bg-muted/30 text-sm text-muted-foreground transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
          isDragActive && "border-ring bg-muted",
          className,
        )}
      >
        <input {...getInputProps()} />
        {previewUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={previewUrl}
            alt="Selected reference"
            className="size-full object-cover"
          />
        ) : fileName ? (
          <span className="max-w-full truncate px-12">{fileName}</span>
        ) : (
          <span className="flex items-center gap-2">
            <ImagePlus className="size-4" />
            {isDragActive ? "Drop image to attach" : label}
          </span>
        )}
        {onClear && (previewUrl || fileName) ? (
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            className="absolute top-2 right-2 bg-background/90"
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
      ) : null}
    </div>
  );
}
