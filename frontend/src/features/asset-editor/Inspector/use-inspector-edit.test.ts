import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  cleanups: [] as Array<() => void>,
  dropzoneOptions: undefined as
    | {
        onDropAccepted: (files: File[]) => void;
        onDropRejected: () => void;
      }
    | undefined,
  uploadFile: vi.fn(),
  stateSetters: [] as ReturnType<typeof vi.fn>[],
  stateValues: [] as unknown[],
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useEffect: (effect: () => void | (() => void)) => {
      const cleanup = effect();
      if (cleanup) mocks.cleanups.push(cleanup);
    },
    useRef: (value: unknown) => ({ current: value }),
    useState: (initial: unknown) => {
      let current =
        mocks.stateValues.length > 0 ? mocks.stateValues.shift() : initial;
      const setter = vi.fn((next: unknown) => {
        current =
          typeof next === "function"
            ? (next as (value: unknown) => unknown)(current)
            : next;
      });
      mocks.stateSetters.push(setter);
      return [current, setter];
    },
  };
});

vi.mock("react-dropzone", () => ({
  useDropzone: (options: typeof mocks.dropzoneOptions) => {
    mocks.dropzoneOptions = options;
    return {
      getInputProps: vi.fn(),
      getRootProps: vi.fn(),
      isDragActive: false,
    };
  },
}));

vi.mock("@/model/upload", () => ({
  uploadFile: mocks.uploadFile,
}));

import { useInspectorEdit } from "./use-inspector-edit";

const animations = [
  { kind: "clip" as const, id: "walk", label: "Walk", frameCount: 4 },
];

const prototype = {
  format: "png-sprite-sheet" as const,
  imageUrl: "/prototype.png",
  frameWidth: 32,
  frameHeight: 32,
  columns: 4,
  rows: 1,
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.cleanups.length = 0;
  mocks.dropzoneOptions = undefined;
  mocks.stateSetters.length = 0;
  mocks.stateValues.length = 0;
  mocks.uploadFile.mockResolvedValue({
    objectKey: "uploads/hero.png",
    objectURL: "https://cdn.example/hero.png?token=signed",
  });
});

describe("useInspectorEdit", () => {
  it("clears the prompt after a successful submission", async () => {
    mocks.stateValues.push(null, null, false, null);
    const onPromptChange = vi.fn();
    const inspector = useInspectorEdit({
      selectedNodes: ["walk"],
      selectedFrames: [],
      prompt: "Refine the walk",
      animations,
      prototype,
      onPromptChange,
      onSubmit: vi.fn().mockResolvedValue(undefined),
    });

    inspector.handleSubmit({ preventDefault: vi.fn() } as never);
    await flushPromises();

    expect(onPromptChange).toHaveBeenCalledWith("");
  });

  it("submits a validated prompt with its reference", async () => {
    const creatingReference = {
      fileName: "hero.png",
      mimeType: "image/png",
      objectKey: "uploads/hero.png",
      previewUrl: "https://cdn.example/hero.png?token=signed",
    };
    mocks.stateValues.push(creatingReference, null, false, null);
    const onPromptChange = vi.fn();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const inspector = useInspectorEdit({
      selectedNodes: ["walk"],
      selectedFrames: [{ nodeId: "walk", index: 1 }],
      prompt: "Refine the walk",
      animations,
      prototype,
      onPromptChange,
      onSubmit,
    });
    const preventDefault = vi.fn();

    inspector.changePrompt("Updated prompt");
    inspector.handleSubmit({ preventDefault } as never);
    await flushPromises();

    expect(inspector.canClearSelection).toBe(true);
    expect(inspector.canSubmit).toBe(true);
    expect(inspector.creatingReference).toBe(creatingReference);
    expect(onPromptChange).toHaveBeenCalledWith("Updated prompt");
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(onSubmit).toHaveBeenCalledWith({
      prompt: "Refine the walk",
      creatingReference,
      target: {
        nodeIds: ["walk"],
        frames: [{ nodeId: "walk", index: 1 }],
      },
    });
  });

  it("supports Enter submission but preserves Shift+Enter", async () => {
    mocks.stateValues.push(null, null, false, null);
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const inspector = useInspectorEdit({
      selectedNodes: [],
      selectedFrames: [],
      prompt: "Edit prototype",
      animations,
      prototype,
      onPromptChange: vi.fn(),
      onSubmit,
    });
    const plainEnter = {
      key: "Enter",
      shiftKey: false,
      preventDefault: vi.fn(),
    };

    inspector.handlePromptKeyDown({
      key: "a",
      shiftKey: false,
      preventDefault: vi.fn(),
    } as never);
    inspector.handlePromptKeyDown({
      key: "Enter",
      shiftKey: true,
      preventDefault: vi.fn(),
    } as never);
    inspector.handlePromptKeyDown(plainEnter as never);
    await flushPromises();

    expect(plainEnter.preventDefault).toHaveBeenCalledOnce();
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("blocks invalid, reading, and externally submitting requests", async () => {
    for (const [prompt, isReadingReference, isSubmitting] of [
      ["", false, false],
      ["Valid", true, false],
      ["Valid", false, true],
    ] as const) {
      mocks.stateValues.push(null, null, isReadingReference, null);
      const onSubmit = vi.fn();
      const inspector = useInspectorEdit({
        selectedNodes: [],
        selectedFrames: [],
        prompt,
        animations,
        prototype,
        onPromptChange: vi.fn(),
        onSubmit,
        isSubmitting,
      });
      inspector.handleSubmit({ preventDefault: vi.fn() } as never);
      await flushPromises();
      expect(inspector.canSubmit).toBe(false);
      expect(onSubmit).not.toHaveBeenCalled();
    }
  });

  it("reads accepted files and reports rejected or unreadable files", async () => {
    mocks.stateValues.push(null, null, false, null);
    useInspectorEdit({
      selectedNodes: [],
      selectedFrames: [],
      prompt: "Prompt",
      animations,
      prototype,
      onPromptChange: vi.fn(),
      onSubmit: vi.fn(),
    });
    const file = { name: "hero.png", type: "image/png" } as File;
    mocks.dropzoneOptions?.onDropAccepted([file]);
    await flushPromises();

    expect(mocks.uploadFile).toHaveBeenCalledWith(
      file,
      expect.any(AbortSignal),
    );
    expect(mocks.stateSetters[0]).toHaveBeenCalledWith({
      fileName: "hero.png",
      mimeType: "image/png",
      objectKey: "uploads/hero.png",
      previewUrl: "https://cdn.example/hero.png?token=signed",
    });

    mocks.dropzoneOptions?.onDropRejected();
    expect(mocks.stateSetters[1]).toHaveBeenCalledWith(
      "Use a PNG, JPEG, or WebP image.",
    );

    mocks.uploadFile.mockRejectedValueOnce(new Error("upload failed"));
    mocks.dropzoneOptions?.onDropAccepted([file]);
    await flushPromises();
    expect(mocks.stateSetters[1]).toHaveBeenCalledWith(
      "We couldn't upload that image. Try again.",
    );
  });

  it("clears and aborts reference work and reports submit failures", async () => {
    mocks.stateValues.push(null, null, false, null);
    const onSubmit = vi.fn().mockRejectedValue(new Error("failed"));
    const inspector = useInspectorEdit({
      selectedNodes: [],
      selectedFrames: [],
      prompt: "Prompt",
      animations,
      prototype,
      onPromptChange: vi.fn(),
      onSubmit,
    });
    const file = { name: "hero.png", type: "image/png" } as File;
    mocks.dropzoneOptions?.onDropAccepted([file]);
    inspector.clearCreatingReference();
    inspector.handleSubmit({ preventDefault: vi.fn() } as never);
    await flushPromises();
    mocks.cleanups.forEach((cleanup) => cleanup());

    expect(mocks.stateSetters[0]).toHaveBeenCalledWith(null);
    expect(mocks.stateSetters[1]).toHaveBeenCalledWith(null);
    expect(mocks.stateSetters[2]).toHaveBeenCalledWith(false);
    expect(mocks.stateSetters[3]).toHaveBeenCalledWith(
      "Unable to send the prompt.",
    );
  });
});

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}
