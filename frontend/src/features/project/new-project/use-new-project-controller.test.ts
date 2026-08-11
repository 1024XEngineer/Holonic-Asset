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
  setters: [] as ReturnType<typeof vi.fn>[],
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
      const setter = vi.fn();
      mocks.setters.push(setter);
      return [typeof initial === "function" ? initial() : initial, setter];
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
vi.mock("@/model/project", () => ({
  projectApi: { generateReference: mocks.generateReference },
}));
vi.mock("../lib/project-context", () => ({
  createNewProjectDraft: () => ({ name: "" }),
  toCreateProjectInput: (value: object) => value,
}));

import { useNewProjectController } from "./use-new-project-controller";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.setters.length = 0;
  mocks.formOptions = undefined;
  mocks.createProject.mockResolvedValue({ id: "project-7" });
  mocks.generateReference.mockResolvedValue("generated.png");
  mocks.navigate.mockResolvedValue(undefined);
  mocks.readFileAsDataUrl.mockResolvedValue("data:image/png;base64,preview");
});

describe("useNewProjectController", () => {
  it("submits a project and navigates to its workspace", async () => {
    useNewProjectController();

    await mocks.formOptions!.onSubmit({ value: { name: "  " } });

    expect(mocks.createProject).toHaveBeenCalledWith({
      name: "Untitled game",
      visualDirection: "",
    });
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "project-7" },
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

  it("resets the form for a chosen start and returns to the library", () => {
    const controller = useNewProjectController();

    controller.start.chooseIdea();
    controller.start.chooseBlank();
    controller.backToLibrary();

    expect(mocks.form.setFieldValue).toHaveBeenCalledWith("reference", "");
    expect(mocks.navigate).toHaveBeenCalledWith({ to: "/projects" });
  });
});
