import { useCallback, useEffect, useMemo } from "react";
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

const EMPTY_PROJECTS: ProjectSummary[] = [];

export function useProjectLibrary(): ProjectLibraryController {
  const navigate = useNavigate({ from: "/projects/" });
  const search = useSearch({ from: "/projects/" });
  const { data: projectData, isSuccess: projectsLoaded } =
    useProjectListQuery();
  const projects = projectData ?? EMPTY_PROJECTS;
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
        search: {},
      }),
    [navigate],
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

  const update = useCallback(
    (updatedProject: ProjectSummary) => updateProject(updatedProject),
    [updateProject],
  );

  const projectModel = useMemo(
    () => ({
      current: project,
      items: projects,
      selectedId: search.project,
      create: createProject,
      remove: removeProject,
      select: selectProject,
      update,
    }),
    [
      project,
      projects,
      search.project,
      createProject,
      removeProject,
      selectProject,
      update,
    ],
  );

  return useMemo(() => ({ project: projectModel }), [projectModel]);
}
