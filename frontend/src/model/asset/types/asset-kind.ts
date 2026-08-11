import { z } from "zod";

export const assetKinds = [
  "character",
  "object",
  "tileset",
  "scenery",
  "uiset",
  "audio",
] as const;

export const assetKindSchema = z.enum(assetKinds);

export type AssetKind = z.infer<typeof assetKindSchema>;

export type CreatableAssetKind = AssetKind;

export const creatableAssetKinds: CreatableAssetKind[] = [...assetKinds];
