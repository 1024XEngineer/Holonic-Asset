import { createFileRoute } from "@tanstack/react-router";
import { ProjectLibraryPage } from "@/pages/projects/project-library-page";

export const Route = createFileRoute("/projects/")({
  component: ProjectLibraryPage,
});
