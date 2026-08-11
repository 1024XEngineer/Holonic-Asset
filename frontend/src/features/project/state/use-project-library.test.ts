import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ProjectSummary } from "@/model/project";

const mocks = vi.hoisted(() => ({
  deleteProject: vi.fn(),
  navigate: vi.fn(),
  projectQuery: {
    data: undefined as ProjectSummary[] | undefined,
    isSuccess: false,
  },
  projectDetailQuery: {
    data: undefined as ProjectSummary | undefined,
  },
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
  mocks.projectQuery.isSuccess = true;
  mocks.projectDetailQuery.data = undefined;
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
    visualDirection: "",
    assetCount: 0,
  };
}
