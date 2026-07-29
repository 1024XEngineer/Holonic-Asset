import {
  addMockAsset,
  copyMockAsset,
  deleteMockAsset,
  listMockAssetGroups,
  saveMockAssetRevision,
} from "./mock";
import {
  getDefaultAssetCanvasSize,
  type AssetKind,
  type ProjectAsset,
} from "@/model";
import { getEnvelope, postEnvelope } from "@/api/fetchers";

/** Matches core-api/internal/dto.AssetListItemResponse. */
export type AssetListItemResponse = {
  assetId: number;
  name: string;
  projectId: number;
  type: "character" | "tileSet" | "audio" | "ui" | "object" | "scenery";
  description: string;
  tags: string[];
  version: number;
};

export type AssetAttributes = Record<string, unknown>;
export type AssetDetailResponse = AssetListItemResponse & {
  attributes: AssetAttributes;
};
export type GetAssetsResponse = { assets: AssetListItemResponse[] };
export type AssetRequest = {
  ID?: number;
  Name: string;
  ProjectID: number;
  Type: AssetListItemResponse["type"];
  Description: string;
  tags?: string[];
  attributes?: AssetAttributes;
  Version?: number;
};
export type CreateAssetResponse = { id: number };
export type CreateAnimationRequest = {
  Name: string;
  AssetID: number;
  Type: AssetListItemResponse["type"];
};
export type CreateAnimationResponse = { id: number };
export type RecordAssetRequest = { AssetID: number };
export type CopyAssetRequest = { AssetID: number };
export type CopyAssetResponse = { newAssetId: number };
export type RollBackAssetRequest = { AssetID: number; Version: number };
export type RollBackAssetResponse = { asset?: AssetRequest };
export type AddTagsRequest = { AssetID: number; Tags: string[] };
export type AddTagsResponse = { tags: string[] };

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

/** HTTP client for supported asset routes. Asset deletion has no backend route. */
export const coreAssetApi = {
  list: (projectID: number) =>
    getEnvelope<GetAssetsResponse>(`/projects/${projectID}/assets`),
  detail: (assetID: number) =>
    getEnvelope<AssetDetailResponse>(`/asset/${assetID}`),
  createCharacter: (asset: AssetRequest) =>
    postEnvelope<CreateAssetResponse>("/asset/characters", { Asset: asset }),
  createObject: (asset: AssetRequest) =>
    postEnvelope<CreateAssetResponse>("/asset/objects", { Asset: asset }),
  createTileset: (asset: AssetRequest) =>
    postEnvelope<CreateAssetResponse>("/asset/tilesets", { Asset: asset }),
  createAnimation: (request: CreateAnimationRequest) =>
    postEnvelope<CreateAnimationResponse>("/asset/animations", request),
  record: (request: RecordAssetRequest) => postEnvelope("/asset/save", request),
  copy: (request: CopyAssetRequest) =>
    postEnvelope<CopyAssetResponse>("/asset/copy", request),
  rollback: (request: RollBackAssetRequest) =>
    postEnvelope<RollBackAssetResponse>("/asset/rollback", request),
  addTags: (request: AddTagsRequest) =>
    postEnvelope<AddTagsResponse>("/asset/tags", request),
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
      tags: item.tags,
      history: [],
      animations: [],
    });
    groups.set(kind, assets);
  }

  return [...groups].map(([kind, assets]) => ({ kind, assets }));
}
