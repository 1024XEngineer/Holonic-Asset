import { createFileRoute } from "@tanstack/react-router";
import { ProjectLibraryPage } from "@/pages/projects/project-library-page";

export const Route = createFileRoute("/projects/")({
  validateSearch: (search: Record<string, unknown>) => ({
    project: typeof search.project === "string" ? search.project : undefined,
    q: typeof search.q === "string" ? search.q : undefined,
  }),
  component: ProjectLibraryPage,
});
