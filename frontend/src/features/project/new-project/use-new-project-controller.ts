import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";

import { useCreateProjectMutation } from "@/model";
import { readFileAsDataUrl } from "@/lib/read-file-as-data-url";
import { projectApi } from "@/model/project";

import {
  createNewProjectDraft,
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
  const [isGeneratingReference, setIsGeneratingReference] = useState(false);
  const [previewError, setPreviewError] = useState<string>();
  const previewReadController = useRef<AbortController | null>(null);
  const [previewMode, setPreviewMode] =
    useState<ProjectPreviewMode>("generate");
  const projectPreview =
    previewMode === "generate" ? generatedPreview : uploadedPreview;

  useEffect(() => () => previewReadController.current?.abort(), []);

  const form = useForm({
    defaultValues: createNewProjectDraft(),
    onSubmit: async ({ value }) => {
      const project = await createProject(
        toCreateProjectInput({
          ...value,
          name: value.name.trim() || "Untitled game",
          reference: projectPreview,
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
    setGeneratedPreview("");
    setUploadedPreview("");
    setPreviewError(undefined);
    form.setFieldValue("reference", "");
    setPreviewMode("generate");
    setStep(1);
    setSelectedStart("idea");
  }, [form]);

  const chooseBlank = useCallback(() => {
    setGeneratedPreview("");
    setUploadedPreview("");
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
    setIsGeneratingReference(true);
    setPreviewError(undefined);
    setPreviewMode("generate");
    setStep(2);
    try {
      const reference = await projectApi.generateReference(
        toCreateProjectInput({
          ...form.state.values,
          name: form.state.values.name.trim() || "Untitled game",
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
    if (selectedStart === "blank") void form.handleSubmit();
    else if (generatedPreview) {
      setPreviewMode("generate");
      setStep(2);
    } else void generateReference();
  }, [form, generateReference, generatedPreview, selectedStart]);

  const returnToStart = useCallback(() => {
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
    form.setFieldValue("reference", uploadedPreview);
  }, [form, uploadedPreview]);

  const generate = useCallback(
    () => void generateReference(),
    [generateReference],
  );

  const setFile = useCallback(
    (file: File) => {
      previewReadController.current?.abort();
      const controller = new AbortController();
      previewReadController.current = controller;
      setUploadedPreview("");
      setPreviewError(undefined);
      void readFileAsDataUrl(file, controller.signal).then(
        (dataUrl) => {
          if (controller.signal.aborted) return;
          setUploadedPreview(dataUrl);
          form.setFieldValue("reference", dataUrl);
        },
        () => {
          if (controller.signal.aborted) return;
          setPreviewError("We couldn't read that image. Try another file.");
        },
      );
    },
    [form],
  );

  const clear = useCallback(() => {
    previewReadController.current?.abort();
    setUploadedPreview("");
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
