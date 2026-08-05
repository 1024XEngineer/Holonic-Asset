import { ProjectLibrary } from "@/features/project";

export function ProjectLibraryPage({ projectId }: { projectId?: string }) {
  return <ProjectLibrary projectId={projectId} />;
}
