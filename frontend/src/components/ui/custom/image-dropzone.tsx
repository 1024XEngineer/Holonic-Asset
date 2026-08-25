/**
 * Custom Component: ImageDropzone
 * Image upload and preview dropzone component built with react-dropzone.
 */

import { ImagePlus } from "lucide-react";
import { useId, type ReactNode } from "react";
import { useDropzone, type Accept, type FileRejection } from "react-dropzone";

import { cn } from "@/lib/utils";

const IMAGE_ACCEPT = {
  "image/jpeg": [".jpg", ".jpeg"],
  "image/png": [".png"],
  "image/webp": [".webp"],
} satisfies Accept;

const INVALID_IMAGE_MESSAGE = "Use a PNG, JPEG, or WebP image.";

export type ImageDropzoneValue = File | string;

export interface ImageDropzoneProps {
  className?: string;
  error?: string;
  label?: string;
  onChange: (value: File | undefined) => void;
  value?: ImageDropzoneValue;
}

export function ImageDropzone({
  className,
  error,
  label = "Upload a reference image",
  onChange,
  value,
}: ImageDropzoneProps) {
  const errorId = useId();
  const previewUrl = typeof value === "string" ? value : undefined;
  const fileName = value instanceof File ? value.name : undefined;
  const hasSelection = Boolean(value);
  const { fileRejections, getInputProps, getRootProps, isDragActive } =
    useDropzone({
      accept: IMAGE_ACCEPT,
      maxFiles: 1,
      multiple: false,
      onDropAccepted: ([file]) => file && onChange(file),
    });
  const message = getRejectionMessage(fileRejections) ?? error;
  let dropzoneContent: ReactNode;
  if (previewUrl) {
    dropzoneContent = (
      <img
        src={previewUrl}
        alt="Selected image"
        className="size-full rounded-[inherit] object-cover"
      />
    );
  } else if (fileName) {
    dropzoneContent = (
      <span className="max-w-full truncate px-12">{fileName}</span>
    );
  } else {
    dropzoneContent = (
      <span className="flex items-center gap-2">
        <ImagePlus className="size-4" />
        {isDragActive ? "Drop image to attach" : label}
      </span>
    );
  }

  return (
    <div className="grid gap-2">
      <div
        {...getRootProps({
          "aria-describedby": message ? errorId : undefined,
          "aria-invalid": message ? true : undefined,
          "aria-label": hasSelection ? "Replace image" : label,
          role: "button",
        })}
        className={cn(
          "group relative flex min-h-28 items-center justify-center overflow-visible rounded-lg border border-dashed bg-muted/30 text-sm text-muted-foreground transition-colors aria-invalid:border-destructive",
          "cursor-pointer hover:bg-muted/60",
          isDragActive && "border-ring bg-muted",
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
        {dropzoneContent}
      </div>
      {message ? (
        <p id={errorId} className="text-xs text-destructive" role="alert">
          {message}
        </p>
      ) : null}
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
