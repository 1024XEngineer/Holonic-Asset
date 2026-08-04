import { createFileRoute } from "@tanstack/react-router";
import { NewProjectPage } from "@/pages/projects/new/new-project-page";

export const Route = createFileRoute("/projects/new")({
  validateSearch: () => ({}),
  component: NewProjectPage,
});
