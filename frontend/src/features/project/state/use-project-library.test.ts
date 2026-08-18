import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ProjectSummary } from "@/model/project";

const mocks = vi.hoisted(() => ({
  deleteProject: vi.fn(),
  navigate: vi.fn(),
  projectQuery: {
    data: undefined as ProjectSummary[] | undefined,
    error: null as Error | null,
    isPending: false,
    isSuccess: false,
    refetch: vi.fn(),
  },
  projectDetailQuery: {
    data: undefined as ProjectSummary | undefined,
    isPending: false,
  },
  rememberedProjectId: undefined as string | undefined,
  reconcile: vi.fn(),
  removeSelection: vi.fn(),
  updateProject: vi.fn(),
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useCallback: (callback: unknown) => callback,
    useEffect: (effect: () => void) => effect(),
    useMemo: (factory: () => unknown) => factory(),
  };
});

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/model", () => ({
  reconcileProjectSelection: mocks.reconcile,
  readLastProjectId: () => mocks.rememberedProjectId,
  removeProjectSelection: mocks.removeSelection,
  useDeleteProjectMutation: () => ({ mutateAsync: mocks.deleteProject }),
  useProjectDetailQuery: () => mocks.projectDetailQuery,
  useProjectListQuery: () => mocks.projectQuery,
  useUpdateProjectMutation: () => ({ mutate: mocks.updateProject }),
}));

import { useProjectLibrary } from "./use-project-library";

const projects = [project("first"), project("second")];

beforeEach(() => {
  vi.clearAllMocks();
  mocks.projectQuery.data = projects;
  mocks.projectQuery.error = null;
  mocks.projectQuery.isPending = false;
  mocks.projectQuery.isSuccess = true;
  mocks.projectQuery.refetch.mockResolvedValue(undefined);
  mocks.projectDetailQuery.data = undefined;
  mocks.projectDetailQuery.isPending = false;
  mocks.rememberedProjectId = undefined;
  mocks.reconcile.mockReturnValue({});
  mocks.removeSelection.mockReturnValue("second");
  mocks.deleteProject.mockResolvedValue(undefined);
  mocks.navigate.mockResolvedValue(undefined);
});

describe("useProjectLibrary", () => {
  it("redirects remembered and missing project selections", () => {
    mocks.reconcile.mockReturnValue({ redirectProjectId: "second" });
    useProjectLibrary(undefined);
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "second" },
      replace: true,
    });

    mocks.reconcile.mockReturnValue({});
    useProjectLibrary("missing");
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/projects",
      replace: true,
    });
  });

  it("loads a remembered project before the project list completes", () => {
    mocks.projectQuery.data = undefined;
    mocks.projectQuery.isSuccess = false;
    mocks.projectDetailQuery.data = projects[1];
    mocks.rememberedProjectId = "second";

    const controller = useProjectLibrary(undefined);

    expect(controller.project.current).toEqual(projects[1]);
    expect(controller.project.selectedId).toBe("second");
    expect(controller.project.isLoading).toBe(false);
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "second" },
      replace: true,
    });
  });

  it("reports project bootstrap loading instead of an unselected state", () => {
    mocks.projectQuery.data = undefined;
    mocks.projectQuery.isPending = true;
    mocks.projectQuery.isSuccess = false;
    mocks.projectDetailQuery.isPending = true;

    const controller = useProjectLibrary(undefined);

    expect(controller.project.current).toBeUndefined();
    expect(controller.project.isLoading).toBe(true);
  });

  it("exposes a failed project list query instead of staying loading", () => {
    const error = new Error("project list failed");
    mocks.projectQuery.data = undefined;
    mocks.projectQuery.error = error;
    mocks.projectQuery.isPending = false;
    mocks.projectQuery.isSuccess = false;

    const controller = useProjectLibrary(undefined);

    expect(controller.project.isLoading).toBe(false);
    expect(controller.project.error).toBe(error);

    controller.project.retry();
    expect(mocks.projectQuery.refetch).toHaveBeenCalledOnce();
  });

  it("keeps cached projects usable when a background refresh fails", () => {
    mocks.projectQuery.error = new Error("project refresh failed");
    mocks.projectQuery.isSuccess = false;

    const controller = useProjectLibrary(undefined);

    expect(controller.project.items).toEqual(projects);
    expect(controller.project.error).toBeUndefined();
    expect(controller.project.isLoading).toBe(false);
  });

  it("shows the unselected state after an empty project list loads", () => {
    mocks.projectQuery.data = [];
    mocks.projectQuery.isSuccess = true;
    mocks.projectDetailQuery.isPending = true;

    const controller = useProjectLibrary(undefined);

    expect(controller.project.current).toBeUndefined();
    expect(controller.project.isLoading).toBe(false);
  });

  it("exposes project navigation, update, and removal actions", async () => {
    const controller = useProjectLibrary("first");

    await controller.project.select("second");
    await controller.project.select(undefined, true);
    await controller.project.create();
    controller.project.update(projects[0]);
    await controller.project.remove("first");

    expect(controller.project.current).toEqual(projects[0]);
    expect(controller.project.items).toEqual(projects);
    expect(mocks.updateProject).toHaveBeenCalledWith(projects[0]);
    expect(mocks.deleteProject).toHaveBeenCalledWith("first");
    expect(mocks.removeSelection).toHaveBeenCalledWith(
      projects,
      "first",
      "first",
    );
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "second" },
      replace: true,
    });
  });

  it("does not navigate after removing a non-selected project", async () => {
    const controller = useProjectLibrary("first");
    mocks.navigate.mockClear();

    await controller.project.remove("second");

    expect(mocks.navigate).not.toHaveBeenCalled();
  });
});

function project(id: string): ProjectSummary {
  return {
    id,
    name: id,
    style: "Top-Down",
    gameType: "Role-playing game",
    platform: "PC",
    description: "",
    reference: "",
    perspective: "Top-Down",
    assetCount: 0,
  };
}
