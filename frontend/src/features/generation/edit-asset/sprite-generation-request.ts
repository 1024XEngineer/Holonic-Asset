import type { CreateGenerationRequest, SpriteAssetKind } from "@/model";

type SpriteEditGenerationRequest = {
  prompt: string;
  creatingReference?: {
    fileName: string;
    mimeType: string;
    objectKey: string;
  };
  target: {
    nodeIds: string[];
    frames: Array<{ nodeId: string; index: number }>;
  };
};

export function buildSpriteGenerationRequest(
  assetKind: SpriteAssetKind,
  assetId: number,
  request: SpriteEditGenerationRequest,
): CreateGenerationRequest {
  const selectedNode = request.target.nodeIds[0];
  const hasSelectedFrames = request.target.frames.length > 0;
  let kind: CreateGenerationRequest["kind"];
  if (hasSelectedFrames) kind = "edit_frames";
  else if (selectedNode && selectedNode !== "prototype")
    kind = "edit_animation";
  else if (assetKind === "character") kind = "edit_character_prototype";
  else kind = "edit_object_prototype";

  let targetAssetPaths: string[] | undefined;
  if (hasSelectedFrames) {
    targetAssetPaths = request.target.frames.map(
      (frame) => `animations.${frame.nodeId}.frames.${frame.index}`,
    );
  } else if (selectedNode === "prototype") {
    targetAssetPaths = ["prototype"];
  } else if (selectedNode) {
    targetAssetPaths = [`animations.${selectedNode}`];
  }

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
