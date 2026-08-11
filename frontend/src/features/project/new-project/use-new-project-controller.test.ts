import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  createProject: vi.fn(),
  form: {
    handleSubmit: vi.fn(),
    setFieldValue: vi.fn(),
    state: { values: { name: "  Moonlit Orchard  " } },
  },
  formOptions: undefined as
    | { onSubmit: (value: { value: object }) => Promise<void> }
    | undefined,
  generateReference: vi.fn(),
  navigate: vi.fn(),
  readFileAsDataUrl: vi.fn(),
  stateIndex: 0,
  stateOverrides: new Map<number, unknown>(),
  setters: [] as ReturnType<typeof vi.fn>[],
  toast: { add: vi.fn() },
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useCallback: (callback: unknown) => callback,
    useEffect: () => undefined,
    useMemo: (factory: () => unknown) => factory(),
    useRef: <T>(value: T) => ({ current: value }),
    useState: (initial: unknown) => {
      const index = mocks.stateIndex++;
      const setter = vi.fn();
      mocks.setters.push(setter);
      const value = mocks.stateOverrides.has(index)
        ? mocks.stateOverrides.get(index)
        : typeof initial === "function"
          ? initial()
          : initial;
      return [value, setter];
    },
  };
});
vi.mock("@tanstack/react-form", () => ({
  useForm: (options: typeof mocks.formOptions) => {
    mocks.formOptions = options;
    return mocks.form;
  },
}));
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mocks.navigate,
}));
vi.mock("@/model", () => ({
  useCreateProjectMutation: () => ({ mutateAsync: mocks.createProject }),
}));
vi.mock("@/lib/read-file-as-data-url", () => ({
  readFileAsDataUrl: mocks.readFileAsDataUrl,
}));
vi.mock("@/components/ui/toast", () => ({ toast: mocks.toast }));
vi.mock("@/model/project", () => ({
  projectApi: { generateReference: mocks.generateReference },
}));
vi.mock("../lib/project-context", () => ({
  createNewProjectDraft: () => ({ name: "" }),
  toCreateBlankProjectInput: (name: string) => ({
    name: name.trim(),
    gameType: "",
    platform: "",
    description: "",
    perspective: "Top-Down",
    reference: "",
    style: "",
  }),
  toCreateProjectInput: (value: object) => value,
}));

import { useNewProjectController } from "./use-new-project-controller";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.stateIndex = 0;
  mocks.stateOverrides.clear();
  mocks.setters.length = 0;
  mocks.formOptions = undefined;
  mocks.createProject.mockResolvedValue({ id: "project-7" });
  mocks.generateReference.mockResolvedValue("generated.png");
  mocks.navigate.mockResolvedValue(undefined);
  mocks.readFileAsDataUrl.mockResolvedValue("data:image/png;base64,preview");
});

describe("useNewProjectController", () => {
  it("requires a project name before submitting", async () => {
    mocks.stateOverrides.set(0, "blank");
    useNewProjectController();

    await mocks.formOptions!.onSubmit({ value: { name: "  " } });

    expect(mocks.toast.add).toHaveBeenCalledWith({
      title: "Project name is required.",
      type: "error",
    });
    expect(mocks.createProject).not.toHaveBeenCalled();
    expect(mocks.navigate).not.toHaveBeenCalled();
  });

  it("submits a named project and navigates to its workspace", async () => {
    useNewProjectController();

    await mocks.formOptions!.onSubmit({
      value: { name: "  Moonlit Orchard  " },
    });

    expect(mocks.createProject).toHaveBeenCalledWith({
      name: "Moonlit Orchard",
      reference: "",
    });
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "project-7" },
    });
  });

  it("submits blank projects without optional context", async () => {
    mocks.stateOverrides.set(0, "blank");
    useNewProjectController();

    await mocks.formOptions!.onSubmit({
      value: {
        name: "  Blank project  ",
        gameType: "Role-playing game",
        platform: "PC",
        description: "Generated default",
        perspective: "Top-Down",
        reference: "reference.png",
      },
    });

    expect(mocks.createProject).toHaveBeenCalledWith({
      name: "Blank project",
      gameType: "",
      platform: "",
      description: "",
      perspective: "Top-Down",
      reference: "",
      style: "",
    });
  });

  it("generates a reference when continuing without a preview", async () => {
    const controller = useNewProjectController();

    controller.form.next();
    await vi.waitFor(() =>
      expect(mocks.generateReference).toHaveBeenCalledOnce(),
    );

    expect(mocks.generateReference).toHaveBeenCalledWith({
      name: "Moonlit Orchard",
    });
    expect(mocks.form.setFieldValue).toHaveBeenCalledWith(
      "reference",
      "generated.png",
    );
  });

  it("reports reference generation failures and always clears the loading state", async () => {
    mocks.generateReference.mockRejectedValueOnce(new Error("offline"));
    const controller = useNewProjectController();

    controller.preview.generate();
    await vi.waitFor(() =>
      expect(mocks.setters[9]).toHaveBeenCalledWith(
        "We couldn't generate that reference. Try again.",
      ),
    );

    expect(mocks.generateReference).toHaveBeenCalledOnce();
    expect(mocks.setters[8]).toHaveBeenLastCalledWith(false);
  });

  it("submits blank projects directly and trims imported game links", async () => {
    mocks.stateOverrides.set(0, "blank");
    const blankController = useNewProjectController();
    blankController.start.chooseBlank();
    blankController.form.next();
    await vi.waitFor(() => expect(mocks.form.handleSubmit).toHaveBeenCalled());

    mocks.stateIndex = 0;
    mocks.setters.length = 0;
    mocks.stateOverrides.clear();
    mocks.stateOverrides.set(3, "link");
    mocks.stateOverrides.set(4, "  https://example.com/game  ");
    const linkController = useNewProjectController();
    linkController.existingGameImport.continue();

    expect(mocks.form.setFieldValue).toHaveBeenLastCalledWith(
      "reference",
      "https://example.com/game",
    );
  });

  it("reads uploaded previews and clears them", async () => {
    const controller = useNewProjectController();
    const file = new File(["preview"], "preview.png", { type: "image/png" });

    controller.preview.setFile(file);
    await vi.waitFor(() =>
      expect(mocks.form.setFieldValue).toHaveBeenCalledWith(
        "reference",
        "data:image/png;base64,preview",
      ),
    );
    controller.preview.clear();

    expect(mocks.readFileAsDataUrl).toHaveBeenCalledWith(
      file,
      expect.any(AbortSignal),
    );
    expect(mocks.form.setFieldValue).toHaveBeenLastCalledWith("reference", "");
  });

  it("keeps the form reference synchronized with the selected preview", async () => {
    mocks.stateOverrides.set(6, "generated.png");
    mocks.stateOverrides.set(7, "uploaded.png");
    mocks.stateOverrides.set(10, "upload");
    const controller = useNewProjectController();

    controller.preview.selectGenerate();
    controller.preview.selectUpload();
    await mocks.formOptions!.onSubmit({ value: { name: "Project" } });

    expect(mocks.form.setFieldValue).toHaveBeenCalledWith(
      "reference",
      "generated.png",
    );
    expect(mocks.form.setFieldValue).toHaveBeenCalledWith(
      "reference",
      "uploaded.png",
    );
    expect(mocks.createProject).toHaveBeenCalledWith({
      name: "Project",
      reference: "uploaded.png",
    });
  });

  it("does not use a local game build filename as an image reference", () => {
    mocks.stateOverrides.set(3, "file");
    mocks.stateOverrides.set(
      5,
      new File(["build"], "game.zip", { type: "application/zip" }),
    );
    const controller = useNewProjectController();

    controller.existingGameImport.continue();

    expect(mocks.form.setFieldValue).toHaveBeenCalledWith("reference", "");
  });

  it("resets the form for a chosen start and returns to the library", () => {
    const controller = useNewProjectController();

    controller.start.chooseIdea();
    controller.start.chooseBlank();
    controller.backToLibrary();

    expect(mocks.form.setFieldValue).toHaveBeenCalledWith("reference", "");
    expect(mocks.navigate).toHaveBeenCalledWith({ to: "/projects" });
  });
});
