import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";

import { useCreateProjectMutation } from "@/model";

import {
  createNewProjectDraft,
  toCreateProjectInput,
} from "../../project-context";

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

function createMockProjectPreview({
  description,
  gameType,
  name,
  visualStyle,
}: {
  description: string;
  gameType: string;
  name: string;
  visualStyle: string;
}) {
  const canvas = document.createElement("canvas");
  canvas.width = 1280;
  canvas.height = 720;
  const context = canvas.getContext("2d");
  if (!context) return "";

  const isPixelArt = /pixel/i.test(visualStyle);
  const palette = isPixelArt
    ? { sky: "#15253e", ground: "#2e704b", detail: "#f0bb52", ui: "#111827" }
    : { sky: "#30516c", ground: "#537d58", detail: "#e6b968", ui: "#1f2937" };
  const projectName = name.trim() || "Untitled game";
  const summary = description.trim() || `${gameType} project overview`;

  context.fillStyle = palette.sky;
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = "#20354c";
  context.fillRect(0, 300, canvas.width, 180);
  context.fillStyle = palette.ground;
  context.fillRect(0, 480, canvas.width, 240);

  for (let x = 0; x < canvas.width; x += 64) {
    context.fillStyle = x % 128 === 0 ? "#3b8658" : "#347a51";
    context.fillRect(x, 560, 62, 58);
  }

  context.fillStyle = palette.detail;
  context.fillRect(570, 360, 140, 180);
  context.fillStyle = "#f5d78f";
  context.fillRect(600, 320, 80, 65);
  context.fillStyle = "#182536";
  context.fillRect(615, 410, 50, 80);

  context.fillStyle = palette.ui;
  context.fillRect(48, 48, 310, 126);
  context.fillStyle = "#ffffff";
  context.font = "600 34px system-ui";
  context.fillText(projectName, 76, 98);
  context.fillStyle = "#cbd5e1";
  context.font = "24px system-ui";
  context.fillText(visualStyle || "Visual direction", 76, 138);

  context.fillStyle = "#ffffff";
  context.font = "28px system-ui";
  context.fillText(summary.slice(0, 76), 48, 670);

  return canvas.toDataURL("image/png");
}
