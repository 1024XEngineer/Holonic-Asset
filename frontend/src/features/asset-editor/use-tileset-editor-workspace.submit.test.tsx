// @vitest-environment happy-dom

import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { AssetWorkspaceData } from "@/model";

const mocks = vi.hoisted(() => ({
  flowSubmit: vi.fn(),
  reportAction: vi.fn(),
}));

vi.mock("./use-editor-generation-workspace", () => ({
  useEditorGenerationWorkspace: () => ({
    snapshot: {
      record: {
        mode: "tileset",
        prompt: "Forest",
        tileset: { gridSize: 4, items: [] },
      },
    },
    candidateRecord: null,
    candidateContent: undefined,
    candidateKind: undefined,
    candidateAnimationId: undefined,
    reviewRun: undefined,
    header: {},
    prompt: "",
    isPromptSubmitting: true,
    isResolvingReview: false,
    setPrompt: vi.fn(),
    submit: mocks.flowSubmit,
    resolveReview: vi.fn(),
    reportAction: mocks.reportAction,
  }),
}));

import { useTilesetEditorWorkspace } from "./use-tileset-editor-workspace";

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.flowSubmit.mockResolvedValue(false);
});

describe("useTilesetEditorWorkspace submission coordination", () => {
  it("does not report a dropped item submission as queued", async () => {
    const { result } = renderHook(() =>
      useTilesetEditorWorkspace({ data: workspace(), onBack: vi.fn() }),
    );

    await act(async () => {
      await result.current?.onGenerateItem({
        itemName: "Oak tree",
        itemDescription: "Old oak",
        shape: [[0, 0]],
        creativeBrief: "Dense leaves",
      });
    });

    expect(mocks.flowSubmit).toHaveBeenCalledTimes(1);
    expect(mocks.reportAction).not.toHaveBeenCalledWith("Oak tree queued");
    expect(result.current?.isGeneratingItem).toBe(true);
  });
});

function workspace(): AssetWorkspaceData {
  return {
    projectName: "Project",
    asset: {
      id: "8",
      projectId: "7",
      kind: "tileset",
      name: "Forest",
      perspective: "Top-Down",
      version: "v1",
      history: [],
    },
    record: {
      mode: "tileset",
      prompt: "Forest",
      tileset: { gridSize: 4, items: [] },
    },
  };
}
