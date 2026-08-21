import type { CreateGenerationRequest, SpriteAssetKind } from "@/model";

import type { InspectorSubmitRequest } from "./Inspector/inspector.types";

export function buildInspectorGenerationRequest(
  assetKind: SpriteAssetKind,
  assetId: number,
  request: InspectorSubmitRequest,
): CreateGenerationRequest {
  const selectedNode = request.target.nodeIds[0];
  const hasSelectedFrames = request.target.frames.length > 0;
  const kind = hasSelectedFrames
    ? "edit_frames"
    : selectedNode && selectedNode !== "prototype"
      ? "edit_animation"
      : assetKind === "character"
        ? "edit_character_prototype"
        : "edit_object_prototype";
  const targetAssetPaths = hasSelectedFrames
    ? request.target.frames.map(
        (frame) => `animations.${frame.nodeId}.frames.${frame.index}`,
      )
    : selectedNode === "prototype"
      ? ["prototype"]
      : selectedNode
        ? [`animations.${selectedNode}`]
        : undefined;

  return {
    assetId,
    kind,
    creative_brief: request.prompt,
    targetAssetPaths,
    parameters: request.creatingReference
      ? {
          creating_reference: request.creatingReference.objectKey,
          creating_reference_file_name: request.creatingReference.fileName,
          creating_reference_mime_type: request.creatingReference.mimeType,
        }
      : undefined,
  };
}
