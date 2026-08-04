import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";

import { useCreateProjectMutation } from "@/model";
import { createMockProjectPreview } from "@/model/project/mock";

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
  const [previewMode, setPreviewMode] =
    useState<ProjectPreviewMode>("generate");
  const projectPreview =
    previewMode === "generate" ? generatedPreview : uploadedPreview;
  const form = useForm({
    defaultValues: createNewProjectDraft(),
    onSubmit: async ({ value }) => {
      const project = await createProject(
        toCreateProjectInput({
          ...value,
          name: value.name.trim() || "Untitled game",
          visualDirection: projectPreview,
        }),
      );
      await navigate({
        to: "/projects",
        search: { project: project.id, q: "" },
      });
    },
  });

  return {
    backToLibrary: () =>
      void navigate({
        to: "/projects",
        search: { project: undefined, q: "" },
      }),
    start: {
      selected: selectedStart,
      chooseIdea: () => {
        setGeneratedPreview("");
        setUploadedPreview("");
        setPreviewMode("generate");
        setStep(1);
        setSelectedStart("idea");
      },
      chooseBlank: () => {
        setGeneratedPreview("");
        setUploadedPreview("");
        setPreviewMode("generate");
        setStep(1);
        setSelectedStart("blank");
      },
      openExistingGameImport: () => setImportOpen(true),
    },
    form: {
      instance: form,
      selectedStart,
      step,
      previous: () => {
        if (step === 1 || selectedStart === "blank") setSelectedStart(null);
        else setStep(1);
      },
      next: () => {
        if (selectedStart === "blank") void form.handleSubmit();
        else {
          setPreviewMode("generate");
          setGeneratedPreview(createMockProjectPreview(form.state.values));
          setStep(2);
        }
      },
      returnToStart: () => {
        setStep(1);
        setSelectedStart(null);
      },
    },
    preview: {
      mode: previewMode,
      url: projectPreview,
      selectGenerate: () => {
        setPreviewMode("generate");
        if (!generatedPreview)
          setGeneratedPreview(createMockProjectPreview(form.state.values));
      },
      selectUpload: () => setPreviewMode("upload"),
      generate: () =>
        setGeneratedPreview(createMockProjectPreview(form.state.values)),
      setFile: (file: File) => {
        const reader = new FileReader();
        reader.onload = () => setUploadedPreview(String(reader.result ?? ""));
        reader.readAsDataURL(file);
      },
      clear: () => setUploadedPreview(""),
    },
    existingGameImport: {
      isOpen: importOpen,
      mode: importMode,
      gameUrl,
      gameFile,
      selectLink: () => setImportMode("link"),
      selectFile: () => setImportMode("file"),
      setGameUrl,
      setGameFile,
      dismiss: () => setImportOpen(false),
      continue: () => {
        setImportOpen(false);
        setStep(1);
        setSelectedStart("existing");
      },
    },
  };
}

export type NewProjectController = ReturnType<typeof useNewProjectController>;
