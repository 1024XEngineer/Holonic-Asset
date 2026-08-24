import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from "react";
import { useDropzone } from "react-dropzone";

import { uploadFile } from "@/model/upload";

import { getInspectorTargetSummary } from "./inspector-target";
import {
  inspectorPromptSchema,
  inspectorSubmitRequestSchema,
  type SpriteInspectorContentProps,
  type InspectorCreatingReference,
} from "./inspector.types";

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
  prototype,
  onPromptChange,
  onSubmit,
  isSubmitting = false,
}: Pick<
  SpriteInspectorContentProps,
  | "selectedNodes"
  | "selectedFrames"
  | "prompt"
  | "animations"
  | "prototype"
  | "onPromptChange"
  | "onSubmit"
  | "isSubmitting"
>) {
  const [creatingReference, setCreatingReference] =
    useState<InspectorCreatingReference | null>(null);
  const [creatingReferenceError, setCreatingReferenceError] = useState<
    string | null
  >(null);
  const [isUploadingCreatingReference, setIsUploadingCreatingReference] =
    useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const creatingReferenceUploadController = useRef<AbortController | null>(
    null,
  );

  useEffect(
    () => () => {
      creatingReferenceUploadController.current?.abort();
    },
    [],
  );

  const clearCreatingReference = () => {
    creatingReferenceUploadController.current?.abort();
    setCreatingReference(null);
    setCreatingReferenceError(null);
    setIsUploadingCreatingReference(false);
  };

  const attachCreatingReference = async (file: File) => {
    creatingReferenceUploadController.current?.abort();
    const controller = new AbortController();
    creatingReferenceUploadController.current = controller;
    setIsUploadingCreatingReference(true);
    setCreatingReferenceError(null);

    try {
      const target = await uploadFile(file, controller.signal);
      if (controller.signal.aborted) return;
      setCreatingReference({
        fileName: file.name,
        mimeType: file.type,
        objectKey: target.objectKey,
        previewUrl: target.objectURL,
      });
    } catch {
      if (!controller.signal.aborted) {
        setCreatingReferenceError("We couldn't upload that image. Try again.");
      }
    } finally {
      if (!controller.signal.aborted) setIsUploadingCreatingReference(false);
    }
  };

  const submit = async () => {
    const result = inspectorSubmitRequestSchema.safeParse({
      prompt,
      creatingReference: creatingReference ?? undefined,
      target: { nodeIds: selectedNodes, frames: selectedFrames },
    });
    if (!result.success || isSubmitting || isUploadingCreatingReference) return;

    setSubmitError(null);
    try {
      await onSubmit(result.data);
      setCreatingReference(null);
      setCreatingReferenceError(null);
    } catch {
      setSubmitError("Unable to send the prompt.");
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
      if (file) void attachCreatingReference(file);
    },
    onDropRejected: () => {
      setCreatingReferenceError("Use a PNG, JPEG, or WebP image.");
    },
  });

  return {
    canClearSelection: selectedNodes.length > 0 || selectedFrames.length > 0,
    canSubmit:
      inspectorPromptSchema.safeParse(prompt).success &&
      !isUploadingCreatingReference &&
      !isSubmitting,
    changePrompt,
    clearCreatingReference,
    dropzone,
    handlePromptKeyDown,
    handleSubmit,
    isUploadingCreatingReference,
    creatingReference,
    creatingReferenceError,
    submitError,
    target: getInspectorTargetSummary(
      selectedNodes,
      selectedFrames,
      animations,
      prototype,
    ),
  };
}
