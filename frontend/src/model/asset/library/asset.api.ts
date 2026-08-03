import {
  addMockAsset,
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
  saveMockAssetRevision,
} from "./mock";
import type { AssetListItemResponse, AssetType } from "./asset.contract";
import {
  getDefaultAssetCanvasSize,
  type AssetKind,
  type ProjectAsset,
} from "@/features/assets/types";

export { coreAssetApi } from "./core-asset.api";
export type {
  AssetDetailResponse,
  AssetListItemResponse,
  AssetMetadataResponse,
  AssetRecordResponse,
  AssetType,
  CopyAssetRequest,
  CopyAssetResponse,
  DeleteAssetRequest,
  DeleteAssetResponse,
  GetAssetRecordsResponse,
  GetAssetsResponse,
  ListAssetsQuery,
  RecordAssetRequest,
  RecordAssetResponse,
  RollBackAssetRequest,
  RollBackAssetResponse,
  UpdateAssetRequest,
  UpdateAssetResponse,
} from "./asset.contract";

export type AssetAttributes = Record<string, unknown>;
export type AssetContentMetadata = Record<string, unknown>;

export type AssetImageResourceResponse = {
  id: number;
  url: string;
};

export type AssetAnimationFrameResponse = AssetImageResourceResponse & {
  duration: number;
};

export type AssetAnimationResponse = {
  id: number;
  name: string;
  frames: AssetAnimationFrameResponse[];
};

type DirectionalAssetContent = {
  viewMode: "side_on" | "top_down";
  directionCount: 1 | 2 | 4 | 8;
  prototype: AssetImageResourceResponse[];
  metadata?: AssetContentMetadata;
};

export type CharacterAssetContent = DirectionalAssetContent & {
  animations: AssetAnimationResponse[];
};

export type ObjectAssetContent = DirectionalAssetContent & {
  animations?: AssetAnimationResponse[];
};

export type TileSetTileResponse = {
  url: string;
  position: { x: number; y: number };
};

export type TileSetItemResponse = {
  name: string;
  tiles: TileSetTileResponse[];
};

export type TileSetAssetContent = {
  tileSize: { width: number; height: number };
  items: TileSetItemResponse[];
  metadata?: AssetContentMetadata;
};

/** Content contracts for these asset types have not been specified yet. */
export type UnspecifiedAssetContent = {
  metadata?: AssetContentMetadata;
  [key: string]: unknown;
};

export type AssetContentByType = {
  character: CharacterAssetContent;
  object: ObjectAssetContent;
  tileSet: TileSetAssetContent;
  audio: UnspecifiedAssetContent;
  ui: UnspecifiedAssetContent;
  scenery: UnspecifiedAssetContent;
};

export type AssetContent = AssetContentByType[AssetType];

export type SaveAssetRevisionInput<Payload> = {
  projectId: string;
  assetId: string;
  description: string;
  payload: Payload;
};

export const assetApi = {
  listGroups: (projectId: string) => listMockAssetGroups(projectId),
  add: (projectId: string, kind: AssetKind, asset: ProjectAsset) =>
    addMockAsset(projectId, kind, asset),
  copy: (projectId: string, assetId: string) =>
    copyMockAsset(projectId, assetId),
  delete: (projectId: string, assetId: string) =>
    deleteMockAsset(projectId, assetId),
  saveRevision: <Payload>({
    projectId,
    assetId,
    description,
    payload,
  }: SaveAssetRevisionInput<Payload>) =>
    saveMockAssetRevision(projectId, assetId, description, payload),
};

export function toAssetGroups(items: AssetListItemResponse[]) {
  const groups = new Map<AssetKind, ProjectAsset[]>();

  for (const item of items) {
    const kind = item.type === "tileSet" ? "tileset" : item.type;
    const assets = groups.get(kind) ?? [];
    assets.push({
      id: String(item.assetId),
      name: item.name,
      description: item.description,
      version: `v${item.version}`,
      canvasSize: getDefaultAssetCanvasSize(kind),
      perspective: "Not specified",
      tags: item.tags ?? [],
      history: [],
      animations: [],
    });
    groups.set(kind, assets);
  }

  return [...groups].map(([kind, assets]) => ({ kind, assets }));
}
