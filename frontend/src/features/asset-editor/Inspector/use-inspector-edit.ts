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
  type EditPromptSubmitRequest,
  type InspectorCreatingReference,
  type InspectorTargetSummary,
  type SpriteInspectorContentProps,
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
  const controller = useEditPrompt({
    prompt,
    onPromptChange,
    onSubmit: async ({ prompt: submittedPrompt, creatingReference }) => {
      const result = inspectorSubmitRequestSchema.safeParse({
        prompt: submittedPrompt,
        creatingReference,
        target: { nodeIds: selectedNodes, frames: selectedFrames },
      });
      if (result.success) await onSubmit(result.data);
    },
    isSubmitting,
    target: getInspectorTargetSummary(
      selectedNodes,
      selectedFrames,
      animations,
      prototype,
    ),
  });

  return {
    ...controller,
    canClearSelection: selectedNodes.length > 0 || selectedFrames.length > 0,
  };
}

export function useEditPrompt({
  prompt,
  onPromptChange,
  onSubmit,
  isSubmitting = false,
  target,
  canSubmitTarget = true,
}: {
  prompt: string;
  onPromptChange: (value: string) => void;
  onSubmit: (request: EditPromptSubmitRequest) => void | Promise<void>;
  isSubmitting?: boolean;
  target: InspectorTargetSummary | null;
  canSubmitTarget?: boolean;
}) {
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
      const upload = await uploadFile(file, controller.signal);
      if (controller.signal.aborted) return;
      setCreatingReference({
        fileName: file.name,
        mimeType: file.type,
        objectKey: upload.objectKey,
        previewUrl: upload.objectURL,
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
    const result = inspectorPromptSchema.safeParse(prompt);
    if (
      !result.success ||
      !canSubmitTarget ||
      isSubmitting ||
      isUploadingCreatingReference
    )
      return;

    setSubmitError(null);
    try {
      await onSubmit({
        prompt: result.data,
        ...(creatingReference ? { creatingReference } : {}),
      });
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
    canSubmit:
      inspectorPromptSchema.safeParse(prompt).success &&
      canSubmitTarget &&
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
    target,
  };
}
