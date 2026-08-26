import { buildTilesetGenerationRequest } from "@/features/generation";
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
  const flow = useEditorGenerationWorkspace<TileSetAssetContent>({
    data,
    onBack,
    toCandidateRecord: toTilesetContentCandidate,
  });

  if (flow.snapshot.record.mode !== "tileset") return null;
  const record = flow.snapshot.record;
  const candidate =
    flow.candidateRecord?.mode === "tileset" ? flow.candidateRecord : null;
  const isTilesetReview =
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
  const reviewItems = record.tileset.items.flatMap((currentItem) => {
    if (!changedItemIds.includes(currentItem.id)) return [];
    const candidateItem = candidateItemsById.get(currentItem.id);
    return candidateItem
      ? [{ itemId: currentItem.id, currentItem, candidateItem }]
      : [];
  });

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
    onResolveReview: (applied: boolean) => void flow.resolveReview(applied),
  };
}
