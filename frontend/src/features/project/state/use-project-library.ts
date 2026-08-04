import { useCallback, useEffect } from "react";
import { useNavigate, useSearch } from "@tanstack/react-router";

import {
  reconcileProjectSelection,
  removeProjectSelection,
  useDeleteProjectMutation,
  useProjectListQuery,
  useUpdateProjectMutation,
} from "@/model";

import type { ProjectSummary } from "@/model/project";

export type ProjectLibraryProjectModel = {
  current?: ProjectSummary;
  items: ProjectSummary[];
  selectedId?: string;
  create: () => Promise<unknown>;
  remove: (projectId: string) => Promise<void>;
  select: (
    projectId: string | undefined,
    replace?: boolean,
  ) => Promise<unknown>;
  update: (project: ProjectSummary) => void;
};

export type ProjectLibraryController = {
  project: ProjectLibraryProjectModel;
};

export function useProjectLibrary(): ProjectLibraryController {
  const navigate = useNavigate({ from: "/projects" });
  const search = useSearch({ from: "/projects" });
  const { data: projects = [], isSuccess: projectsLoaded } =
    useProjectListQuery();
  const { mutateAsync: deleteProject } = useDeleteProjectMutation();
  const { mutate: updateProject } = useUpdateProjectMutation();
  const project = projects.find((item) => item.id === search.project);

  const selectProject = useCallback(
    (projectId: string | undefined, replace = false) =>
      navigate({
        to: "/projects",
        search: { project: projectId, q: "" },
        replace,
      }),
    [navigate],
  );

  useEffect(() => {
    if (!projectsLoaded) return;
    const selection = reconcileProjectSelection(projects, search.project);
    if (selection.redirectProjectId)
      void selectProject(selection.redirectProjectId, true);
  }, [projects, projectsLoaded, search.project, selectProject]);

  const createProject = useCallback(
    () =>
      navigate({
        to: "/projects/new",
        search: { project: search.project, q: search.q },
      }),
    [navigate, search.project, search.q],
  );

  const removeProject = useCallback(
    async (projectId: string) => {
      await deleteProject(projectId);
      const fallbackProjectId = removeProjectSelection(
        projects,
        projectId,
        search.project,
      );
      if (search.project === projectId)
        await selectProject(fallbackProjectId, true);
    },
    [deleteProject, projects, search.project, selectProject],
  );

  return {
    project: {
      current: project,
      items: projects,
      selectedId: search.project,
      create: createProject,
      remove: removeProject,
      select: selectProject,
      update: (updatedProject: ProjectSummary) => updateProject(updatedProject),
    },
  };
}
