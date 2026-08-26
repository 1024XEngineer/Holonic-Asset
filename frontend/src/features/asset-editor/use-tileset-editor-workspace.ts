import { useMemo, useState } from "react";

import {
  buildAddTilesetItemGenerationRequest,
  buildTilesetGenerationRequest,
  type CreateTilesetItemRequest,
  type GenerationTaskListItem,
} from "@/features/generation";
import {
  getTilesetCandidateItemIds,
  toTilesetContentCandidate,
  type AssetWorkspaceData,
  type TileSetAssetContent,
} from "@/model";

import type { EditPromptSubmitRequest } from "./Inspector/inspector.types";
import type { ResolvedTilesetEditTarget } from "./tileset-edit-target";
import { useEditorGenerationWorkspace } from "./use-editor-generation-workspace";

export function useTilesetEditorWorkspace({
  data,
  onBack,
}: {
  data: AssetWorkspaceData;
  onBack: () => void;
}) {
  const [itemTask, setItemTask] = useState<GenerationTaskListItem | null>(null);
  const additionalTasks = useMemo(
    () => (itemTask ? [itemTask] : []),
    [itemTask],
  );
  const flow = useEditorGenerationWorkspace<TileSetAssetContent>({
    data,
    onBack,
    toCandidateRecord: toTilesetContentCandidate,
    additionalTasks,
    isAdditionalGenerationPending: itemTask?.status === "processing",
  });

  if (flow.snapshot.record.mode !== "tileset") return null;
  const record = flow.snapshot.record;
  const candidate =
    flow.candidateRecord?.mode === "tileset" ? flow.candidateRecord : null;
  const { reportAction } = flow;
  const isTilesetReview =
    flow.candidateKind === "add_tileset_item" ||
    flow.candidateKind === "edit_tileset_item" ||
    flow.candidateKind === "edit_tiles";
  const changedItemIds = getTilesetCandidateItemIds(
    flow.candidateContent,
    record.tileset.items,
    record.tileset.gridSize,
  );
  const candidateItemsById = new Map(
    candidate?.tileset.items.map((item) => [item.id, item]),
  );
  const reviewItems = [
    ...record.tileset.items.flatMap((currentItem) => {
      if (!changedItemIds.includes(currentItem.id)) return [];
      const candidateItem = candidateItemsById.get(currentItem.id);
      return candidateItem
        ? [
            {
              kind: "comparison" as const,
              itemId: currentItem.id,
              currentItem,
              candidateItem,
            },
          ]
        : [];
    }),
    ...changedItemIds.flatMap((itemId) => {
      if (record.tileset.items.some((item) => item.id === itemId)) return [];
      const candidateItem = candidateItemsById.get(itemId);
      return candidateItem
        ? [{ kind: "new-item" as const, itemId, candidateItem }]
        : [];
    }),
  ];

  const submit = async (
    request: EditPromptSubmitRequest,
    target: ResolvedTilesetEditTarget,
  ) => {
    const projectId = Number(data.asset.projectId);
    const assetId = Number(data.asset.id);
    if (
      !Number.isSafeInteger(projectId) ||
      projectId <= 0 ||
      !Number.isSafeInteger(assetId) ||
      assetId <= 0
    ) {
      throw new Error("Tileset editing requires persisted identifiers.");
    }
    await flow.submit({
      prompt: request.prompt,
      request: buildTilesetGenerationRequest({
        assetId,
        prompt: request.prompt,
        creatingReference: request.creatingReference,
        target,
      }),
    });
  };

  const generateItem = async (request: CreateTilesetItemRequest) => {
    const projectId = Number(data.asset.projectId);
    const assetId = Number(data.asset.id);
    if (
      !Number.isSafeInteger(projectId) ||
      projectId <= 0 ||
      !Number.isSafeInteger(assetId) ||
      assetId <= 0
    ) {
      throw new Error(
        "Tileset item generation requires persisted identifiers.",
      );
    }
    const taskId = `tileset-item-${crypto.randomUUID()}`;
    setItemTask({
      id: taskId,
      name: request.itemName,
      prompt: request.creativeBrief,
      status: "processing",
    });
    try {
      const submitted = await flow.submit({
        prompt: request.creativeBrief,
        request: buildAddTilesetItemGenerationRequest({ assetId, request }),
      });
      if (submitted) reportAction(`${request.itemName} queued`);
    } catch {
      reportAction("Tileset item generation failed");
    } finally {
      setItemTask((current) => (current?.id === taskId ? null : current));
    }
  };

  return {
    header: flow.header,
    gridSize: record.tileset.gridSize,
    sourceItems: record.tileset.items,
    items:
      isTilesetReview && candidate
        ? candidate.tileset.items
        : record.tileset.items,
    history: data.asset.history,
    prompt: flow.prompt,
    isSubmitting: flow.isPromptSubmitting,
    review:
      isTilesetReview && candidate
        ? {
            items: reviewItems,
            isResolving: flow.isResolvingReview,
          }
        : undefined,
    onPromptChange: flow.setPrompt,
    onSubmit: submit,
    onGenerateItem: (request: CreateTilesetItemRequest) =>
      generateItem(request),
    isGeneratingItem: itemTask !== null || flow.isPromptSubmitting,
    onResolveReview: (applied: boolean) => void flow.resolveReview(applied),
  };
}
