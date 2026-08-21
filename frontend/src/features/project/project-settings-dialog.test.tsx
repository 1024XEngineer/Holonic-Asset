// @vitest-environment happy-dom

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import type { ProjectSummary } from "@/model/project";
import { withI18n } from "@/testing/with-i18n";

const mocks = vi.hoisted(() => ({
  regenerateReference: vi.fn(),
  uploadFile: vi.fn(),
}));

vi.mock("@/model/project", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/model/project")>();
  return {
    ...actual,
    projectApi: {
      ...actual.projectApi,
      regenerateReference: mocks.regenerateReference,
    },
  };
});

vi.mock("@/model/upload", () => ({
  uploadFile: mocks.uploadFile,
}));

import { ProjectSettingsDialog } from "./project-settings-dialog";

const uploadedReference =
  "https://cdn.example/uploads/reference.png?token=signed";
const project: ProjectSummary = {
  id: "project-7",
  name: "Moonlit Orchard",
  gameType: "Role-playing game",
  platform: "PC",
  description: "Restore the orchard.",
  reference: "https://cdn.example/uploads/original.png?token=old",
  style: "Top-Down",
  perspective: "Top-Down",
};

beforeAll(() => {
  Object.defineProperty(Element.prototype, "getAnimations", {
    configurable: true,
    value: () => [],
  });
});

beforeEach(() => {
  mocks.uploadFile.mockResolvedValue({
    objectKey: "uploads/reference.png",
    objectURL: uploadedReference,
  });
  mocks.regenerateReference.mockResolvedValue(
    "https://cdn.example/uploads/generated.png?token=fresh",
  );
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ProjectSettingsDialog", () => {
  it("regenerates an uploaded reference with its URL instead of its object key", async () => {
    render(
      withI18n(
        <ProjectSettingsDialog
          project={project}
          onOpenChange={vi.fn()}
          onSave={vi.fn()}
        />,
      ),
    );

    const fileInput =
      document.querySelector<HTMLInputElement>('input[type="file"]');
    if (!fileInput) throw new Error("Expected the reference file input.");

    const file = new File(["reference"], "reference.png", {
      type: "image/png",
    });
    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() =>
      expect(mocks.uploadFile).toHaveBeenCalledWith(
        file,
        expect.any(AbortSignal),
      ),
    );
    await waitFor(() =>
      expect(screen.getByAltText("Reference image").getAttribute("src")).toBe(
        uploadedReference,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Regenerate preview" }));

    await waitFor(() =>
      expect(mocks.regenerateReference).toHaveBeenCalledWith({
        ...project,
        reference: uploadedReference,
      }),
    );
  });
});
