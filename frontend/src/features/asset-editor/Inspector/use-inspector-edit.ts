import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from "react";
import { useDropzone } from "react-dropzone";

import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";

import { getInspectorTargetSummary } from "./inspector-target";
import type { InspectorEditProps, InspectorReference } from "./inspector.types";

const imageAccept = {
  "image/jpeg": [".jpg", ".jpeg"],
  "image/png": [".png"],
  "image/webp": [".webp"],
};

export function useInspectorEdit({
  selectedNodes,
  selectedFrames,
  prompt,
  animations,
  onPromptChange,
  onSubmit,
  isSubmitting = false,
}: Pick<
  InspectorEditProps,
  | "selectedNodes"
  | "selectedFrames"
  | "prompt"
  | "animations"
  | "onPromptChange"
  | "onSubmit"
  | "isSubmitting"
>) {
  const [reference, setReference] = useState<InspectorReference | null>(null);
  const [referenceError, setReferenceError] = useState<string | null>(null);
  const [isReadingReference, setIsReadingReference] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const referenceReadController = useRef<AbortController | null>(null);

  useEffect(
    () => () => {
      referenceReadController.current?.abort();
    },
    [],
  );

  const clearReference = () => {
    referenceReadController.current?.abort();
    setReference(null);
    setReferenceError(null);
    setIsReadingReference(false);
  };

  const attachReference = async (file: File) => {
    referenceReadController.current?.abort();
    const controller = new AbortController();
    referenceReadController.current = controller;
    setIsReadingReference(true);
    setReferenceError(null);

    try {
      const dataUrl = await readFileAsDataUrl(file, controller.signal);
      if (controller.signal.aborted) return;
      setReference({ fileName: file.name, mimeType: file.type, dataUrl });
    } catch {
      if (!controller.signal.aborted) {
        setReferenceError("We couldn't read that image. Try another file.");
      }
    } finally {
      if (!controller.signal.aborted) setIsReadingReference(false);
    }
  };

  const submit = async () => {
    const normalizedPrompt = prompt.trim();
    if (!normalizedPrompt || isSubmitting || isReadingReference) return;

    setSubmitError(null);
    try {
      await onSubmit({
        prompt: normalizedPrompt,
        reference: reference ?? undefined,
        target: { nodeIds: selectedNodes, frames: selectedFrames },
      });
      setReference(null);
      setReferenceError(null);
    } catch (error) {
      setSubmitError(
        error instanceof Error && error.message.trim()
          ? error.message
          : "Unable to send the prompt.",
      );
    }
  };

  const changePrompt = (value: string) => {
    setSubmitError(null);
    onPromptChange(value);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void submit();
  };

  const handlePromptKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key !== "Enter" || event.shiftKey) return;
    event.preventDefault();
    void submit();
  };

  const dropzone = useDropzone({
    accept: imageAccept,
    maxFiles: 1,
    multiple: false,
    noClick: true,
    noKeyboard: true,
    onDropAccepted: ([file]) => {
      if (file) void attachReference(file);
    },
    onDropRejected: () => {
      setReferenceError("Use a PNG, JPEG, or WebP image.");
    },
  });

  return {
    canClearSelection: selectedNodes.length > 0 || selectedFrames.length > 0,
    canSubmit: Boolean(prompt.trim()) && !isReadingReference && !isSubmitting,
    changePrompt,
    clearReference,
    dropzone,
    handlePromptKeyDown,
    handleSubmit,
    isReadingReference,
    reference,
    referenceError,
    submitError,
    target: getInspectorTargetSummary(
      selectedNodes,
      selectedFrames,
      animations,
    ),
  };
}
