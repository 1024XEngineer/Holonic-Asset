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

export type AssetType =
  | "character"
  | "object"
  | "tileSet"
  | "audio"
  | "ui"
  | "scenery";

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

/** Matches core-api/internal/dto.AssetListItemResponse. */
export type AssetListItemResponse<Type extends AssetType = AssetType> = {
  assetId: number;
  name: string;
  projectId: number;
  type: Type;
  description: string;
  tags: string[];
  version: number;
};

export type AssetMetadataResponse<Type extends AssetType = AssetType> =
  AssetListItemResponse<Type> & {
    attributes: AssetAttributes;
  };

type AssetDetailResponseByType = {
  [Type in AssetType]: AssetMetadataResponse<Type> & {
    content: AssetContentByType[Type];
  };
};

export type AssetDetailResponse<Type extends AssetType = AssetType> =
  AssetDetailResponseByType[Type];

export type ListAssetsQuery = {
  query?: string;
  tags?: string[];
  types?: AssetType[];
};

export type UpdateAssetRequest = {
  assetId: number;
  name?: string;
  projectId?: number;
  type?: AssetType;
  description?: string;
  tags?: string[];
  attributes?: AssetAttributes;
  version?: number;
};

export type UpdateAssetResponse = AssetMetadataResponse;

export type AssetRecordResponse<Content extends AssetContent = AssetContent> = {
  recordId: number;
  assetId: number;
  version: number;
  contentId: number;
  createdAt: string;
  content: Content;
};

export type GetAssetsResponse = { assets: AssetListItemResponse[] };
export type AssetRequest = {
  ID?: number;
  Name: string;
  ProjectID: number;
  Type: AssetType;
  Description: string;
  tags?: string[];
  attributes?: AssetAttributes;
  Version?: number;
};
export type CreateAssetResponse = { id: number };
export type CreateAnimationRequest = {
  Name: string;
  AssetID: number;
  Type: AssetType;
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
