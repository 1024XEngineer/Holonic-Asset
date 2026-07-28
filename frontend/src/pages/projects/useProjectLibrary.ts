import { useCallback, useEffect } from "react";
import { useNavigate, useSearch } from "@tanstack/react-router";

import {
  useAssetLibraryQuery,
  useCopyAssetMutation,
  useDeleteAssetMutation,
  useEnqueueGenerationMutation,
  useGenerationRunsQuery,
  reconcileProjectSelection,
  removeProjectSelection,
  useDeleteProjectMutation,
  useProjectListQuery,
  useUpdateProjectMutation,
} from "@/api";
import type { CreationRequest, ProjectSummary } from "@/model";

export function useProjectLibrary() {
  const navigate = useNavigate({ from: "/projects" });
  const search = useSearch({ from: "/projects" });
  const { data: projects = [], isSuccess: projectsLoaded } =
    useProjectListQuery();
  const { data: assetGroups = [] } = useAssetLibraryQuery(search.project);
  const { data: runs = [] } = useGenerationRunsQuery(search.project);
  const { mutate: copyAsset } = useCopyAssetMutation();
  const { mutate: deleteAsset } = useDeleteAssetMutation();
  const { mutate: enqueueRun } = useEnqueueGenerationMutation();
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

  const openAsset = useCallback(
    (assetId: string) => {
      if (project) {
        void navigate({
          to: "/projects/$projectId/assets/$assetId",
          params: { projectId: project.id, assetId },
          search: { project: search.project, q: search.q },
        });
      }
    },
    [navigate, project, search.project, search.q],
  );

  const changeQuery = useCallback(
    (q: string) =>
      navigate({
        to: "/projects",
        search: { project: search.project, q },
        replace: true,
      }),
    [navigate, search.project],
  );

  const createAsset = useCallback(
    (request: CreationRequest) => {
      if (project) enqueueRun({ projectId: project.id, request });
    },
    [enqueueRun, project],
  );

  const copyProjectAsset = useCallback(
    (assetId: string) => {
      if (project) copyAsset({ projectId: project.id, assetId });
    },
    [copyAsset, project],
  );

  const deleteProjectAsset = useCallback(
    (assetId: string) => {
      if (project) deleteAsset({ projectId: project.id, assetId });
    },
    [deleteAsset, project],
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
    assetLibrary: {
      groups: assetGroups,
      query: search.q ?? "",
      changeQuery,
      copyAsset: copyProjectAsset,
      createAsset,
      deleteAsset: deleteProjectAsset,
      openAsset,
    },
    generation: { runs },
  };
}
