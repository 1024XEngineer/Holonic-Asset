import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";

import { useCreateProjectMutation } from "@/model";
import { toast } from "@/components/ui/toast";
import { projectApi } from "@/model/project";
import { uploadFile } from "@/model/upload";

import {
  createNewProjectDraft,
  toCreateBlankProjectInput,
  toCreateProjectInput,
} from "../lib/project-context";

type NewProjectStart = "idea" | "blank" | "existing" | null;
type ExistingGameImportMode = "link" | "file";
type NewProjectStep = 1 | 2;
type ProjectPreviewMode = "generate" | "upload";

export function useNewProjectController() {
  const navigate = useNavigate({ from: "/projects/new" });
  const { mutateAsync: createProject } = useCreateProjectMutation();
  const [selectedStart, setSelectedStart] = useState<NewProjectStart>(null);
  const [step, setStep] = useState<NewProjectStep>(1);
  const [importOpen, setImportOpen] = useState(false);
  const [importMode, setImportMode] = useState<ExistingGameImportMode>("link");
  const [gameUrl, setGameUrl] = useState("");
  const [gameFile, setGameFile] = useState<File | null>(null);
  const [generatedPreview, setGeneratedPreview] = useState("");
  const [uploadedPreview, setUploadedPreview] = useState("");
  const [uploadedReference, setUploadedReference] = useState("");
  const [isGeneratingReference, setIsGeneratingReference] = useState(false);
  const [isUploadingReference, setIsUploadingReference] = useState(false);
  const [previewError, setPreviewError] = useState<string>();
  const previewUploadController = useRef<AbortController | null>(null);
  const [previewMode, setPreviewMode] =
    useState<ProjectPreviewMode>("generate");
  const projectPreview =
    previewMode === "generate" ? generatedPreview : uploadedPreview;
  const reference =
    previewMode === "generate" ? generatedPreview : uploadedReference;

  useEffect(() => () => previewUploadController.current?.abort(), []);

  const form = useForm({
    defaultValues: createNewProjectDraft(),
    onSubmit: async ({ value }) => {
      const name = requireProjectName(value.name);
      if (!name) return;

      const project = await createProject(
        selectedStart === "blank"
          ? toCreateBlankProjectInput(name)
          : toCreateProjectInput({
              ...value,
              name,
              reference,
            }),
      );
      await navigate({
        to: "/projects/$projectId",
        params: { projectId: project.id },
      });
    },
  });

  const backToLibrary = useCallback(
    () =>
      void navigate({
        to: "/projects",
      }),
    [navigate],
  );

  const chooseIdea = useCallback(() => {
    previewUploadController.current?.abort();
    previewUploadController.current = null;
    setGeneratedPreview("");
    setUploadedPreview("");
    setUploadedReference("");
    setIsUploadingReference(false);
    setPreviewError(undefined);
    form.setFieldValue("reference", "");
    setPreviewMode("generate");
    setStep(1);
    setSelectedStart("idea");
  }, [form]);

  const chooseBlank = useCallback(() => {
    previewUploadController.current?.abort();
    previewUploadController.current = null;
    setGeneratedPreview("");
    setUploadedPreview("");
    setUploadedReference("");
    setIsUploadingReference(false);
    setPreviewError(undefined);
    form.setFieldValue("reference", "");
    setPreviewMode("generate");
    setStep(1);
    setSelectedStart("blank");
  }, [form]);

  const openExistingGameImport = useCallback(() => setImportOpen(true), []);

  const previous = useCallback(() => {
    if (step === 1 || selectedStart === "blank") setSelectedStart(null);
    else setStep(1);
  }, [selectedStart, step]);

  const generateReference = useCallback(async () => {
    if (isGeneratingReference) return;
    const name = requireProjectName(form.state.values.name);
    if (!name) return;

    setIsGeneratingReference(true);
    setPreviewError(undefined);
    setPreviewMode("generate");
    setStep(2);
    try {
      const reference = await projectApi.generateReference(
        toCreateProjectInput({
          ...form.state.values,
          name,
        }),
      );
      form.setFieldValue("reference", reference);
      setGeneratedPreview(reference);
    } catch {
      setPreviewError("We couldn't generate that reference. Try again.");
    } finally {
      setIsGeneratingReference(false);
    }
  }, [form, isGeneratingReference]);

  const next = useCallback(() => {
    if (!requireProjectName(form.state.values.name)) return;
    if (selectedStart === "blank") void form.handleSubmit();
    else if (generatedPreview) {
      setPreviewMode("generate");
      setStep(2);
    } else void generateReference();
  }, [form, generateReference, generatedPreview, selectedStart]);

  const returnToStart = useCallback(() => {
    previewUploadController.current?.abort();
    previewUploadController.current = null;
    setIsUploadingReference(false);
    setStep(1);
    setSelectedStart(null);
  }, []);

  const selectGenerate = useCallback(() => {
    setPreviewMode("generate");
    form.setFieldValue("reference", generatedPreview);
    if (!generatedPreview) void generateReference();
  }, [form, generateReference, generatedPreview]);

  const selectUpload = useCallback(() => {
    setPreviewMode("upload");
    form.setFieldValue("reference", uploadedReference);
  }, [form, uploadedReference]);

  const generate = useCallback(
    () => void generateReference(),
    [generateReference],
  );

  const setFile = useCallback(
    (file: File) => {
      previewUploadController.current?.abort();
      const controller = new AbortController();
      previewUploadController.current = controller;
      setUploadedPreview("");
      setUploadedReference("");
      form.setFieldValue("reference", "");
      setIsUploadingReference(true);
      setPreviewError(undefined);
      void (async () => {
        try {
          const target = await uploadFile(file, controller.signal);
          if (controller.signal.aborted) return;
          setUploadedPreview(target.objectURL);
          setUploadedReference(target.objectKey);
        } catch {
          if (controller.signal.aborted) return;
          setPreviewError("We couldn't upload that image. Try again.");
        } finally {
          if (previewUploadController.current === controller) {
            previewUploadController.current = null;
            setIsUploadingReference(false);
          }
        }
      })();
    },
    [form],
  );

  const clear = useCallback(() => {
    previewUploadController.current?.abort();
    previewUploadController.current = null;
    setUploadedPreview("");
    setUploadedReference("");
    setIsUploadingReference(false);
    form.setFieldValue("reference", "");
    setPreviewError(undefined);
  }, [form]);

  const selectLink = useCallback(() => setImportMode("link"), []);
  const selectFile = useCallback(() => setImportMode("file"), []);

  const continueExistingGameImport = useCallback(() => {
    form.setFieldValue(
      "reference",
      importMode === "link" ? gameUrl.trim() : "",
    );
    setImportOpen(false);
    setStep(1);
    setSelectedStart("existing");
  }, [form, gameUrl, importMode]);

  const start = useMemo(
    () => ({
      selected: selectedStart,
      chooseIdea,
      chooseBlank,
      openExistingGameImport,
    }),
    [chooseBlank, chooseIdea, openExistingGameImport, selectedStart],
  );

  const formController = useMemo(
    () => ({
      instance: form,
      selectedStart,
      step,
      isGenerating: isGeneratingReference,
      previous,
      next,
      returnToStart,
    }),
    [
      form,
      isGeneratingReference,
      next,
      previous,
      returnToStart,
      selectedStart,
      step,
    ],
  );

  const preview = useMemo(
    () => ({
      mode: previewMode,
      url: projectPreview,
      isGenerating: isGeneratingReference,
      isUploading: isUploadingReference,
      selectGenerate,
      selectUpload,
      generate,
      setFile,
      error: previewError,
      clear,
    }),
    [
      clear,
      generate,
      isGeneratingReference,
      isUploadingReference,
      previewError,
      previewMode,
      projectPreview,
      selectGenerate,
      selectUpload,
      setFile,
    ],
  );

  const existingGameImport = useMemo(
    () => ({
      isOpen: importOpen,
      mode: importMode,
      gameUrl,
      gameFile,
      selectLink,
      selectFile,
      setGameUrl,
      setGameFile,
      dismiss: () => setImportOpen(false),
      continue: continueExistingGameImport,
    }),
    [
      continueExistingGameImport,
      gameFile,
      gameUrl,
      importMode,
      importOpen,
      selectFile,
      selectLink,
    ],
  );

  return useMemo(
    () => ({
      backToLibrary,
      start,
      form: formController,
      preview,
      existingGameImport,
    }),
    [backToLibrary, existingGameImport, formController, preview, start],
  );
}

export type NewProjectController = ReturnType<typeof useNewProjectController>;

function requireProjectName(name: string) {
  const value = name.trim();
  if (value) return value;

  toast.add({ title: "Project name is required.", type: "error" });
  return undefined;
}
