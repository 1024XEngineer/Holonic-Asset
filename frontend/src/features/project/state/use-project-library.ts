import { useCallback, useEffect, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";

import {
  reconcileProjectSelection,
  readLastProjectId,
  removeProjectSelection,
  useDeleteProjectMutation,
  useProjectDetailQuery,
  useProjectListQuery,
  useUpdateProjectMutation,
} from "@/model";

import type { ProjectSummary } from "@/model/project";

export type ProjectLibraryProjectModel = {
  current?: ProjectSummary;
  isLoading: boolean;
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

export function useProjectLibrary(
  selectedProjectId?: string,
): ProjectLibraryController {
  const navigate = useNavigate();
  const { data: projectData, isSuccess: projectsLoaded } =
    useProjectListQuery();
  const rememberedProjectId = useMemo(
    () => (selectedProjectId ? undefined : readLastProjectId()),
    [selectedProjectId],
  );
  const requestedProjectId = selectedProjectId ?? rememberedProjectId;
  const { data: projectDetail, isPending: projectDetailPending } =
    useProjectDetailQuery(requestedProjectId);
  const projects = projectData ?? EMPTY_PROJECTS;
  const { mutateAsync: deleteProject } = useDeleteProjectMutation();
  const { mutate: updateProject } = useUpdateProjectMutation();
  const rememberedProjectExists = projects.some(
    (item) => item.id === rememberedProjectId,
  );
  const effectiveProjectId =
    selectedProjectId ??
    (!projectsLoaded || projectDetail || rememberedProjectExists
      ? rememberedProjectId
      : undefined);
  const project =
    projectDetail ?? projects.find((item) => item.id === effectiveProjectId);

  const selectProject = useCallback(
    (projectId: string | undefined, replace = false) => {
      if (projectId)
        return navigate({
          to: "/projects/$projectId",
          params: { projectId },
          replace,
        });

      return navigate({ to: "/projects", replace });
    },
    [navigate],
  );

  useEffect(() => {
    if (!selectedProjectId && rememberedProjectId) {
      if (projectDetail || (projectsLoaded && rememberedProjectExists)) {
        void selectProject(rememberedProjectId, true);
      } else if (projectsLoaded) {
        reconcileProjectSelection(projects, undefined);
      }
      return;
    }
    if (!projectsLoaded) return;
    const selection = reconcileProjectSelection(projects, selectedProjectId);
    if (selection.redirectProjectId)
      void selectProject(selection.redirectProjectId, true);
    else if (selectedProjectId && !project) void selectProject(undefined, true);
  }, [
    project,
    projectDetail,
    projects,
    projectsLoaded,
    rememberedProjectExists,
    rememberedProjectId,
    selectedProjectId,
    selectProject,
  ]);

  const createProject = useCallback(
    () =>
      navigate({
        to: "/projects/new",
      }),
    [navigate],
  );

  const removeProject = useCallback(
    async (projectId: string) => {
      await deleteProject(projectId);
      const fallbackProjectId = removeProjectSelection(
        projects,
        projectId,
        selectedProjectId,
      );
      if (selectedProjectId === projectId)
        await selectProject(fallbackProjectId, true);
    },
    [deleteProject, projects, selectedProjectId, selectProject],
  );

  const update = useCallback(
    (updatedProject: ProjectSummary) => updateProject(updatedProject),
    [updateProject],
  );

  const projectModel = useMemo(
    () => ({
      current: project,
      isLoading:
        !project &&
        (!projectsLoaded ||
          Boolean(effectiveProjectId && projectDetailPending)),
      items: projects,
      selectedId: effectiveProjectId,
      create: createProject,
      remove: removeProject,
      select: selectProject,
      update,
    }),
    [
      project,
      projectDetailPending,
      projects,
      projectsLoaded,
      effectiveProjectId,
      createProject,
      removeProject,
      selectProject,
      update,
    ],
  );

  return useMemo(() => ({ project: projectModel }), [projectModel]);
}
