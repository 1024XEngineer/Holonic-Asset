import { projectApi } from "../../project";
import { coreAssetApi } from "../library/core-asset.api";
import type {
  AssetRecord,
  AssetRecordSaveResult,
  AssetWorkspaceData,
  GetAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
import { toCoreAssetHistory } from "./core-asset-workspace";
import {
  toCoreSceneryAssetContent,
  toCoreSceneryAssetWorkspace,
} from "./core-scenery-record";
import {
  toCoreSpriteAssetContent,
  toCoreSpriteAssetWorkspace,
} from "./core-sprite-record";
import {
  toCoreTilesetAssetContent,
  toCoreTilesetAssetWorkspace,
} from "./core-tileset-record";
import {
  toCoreUISetAssetContent,
  toCoreUISetAssetWorkspace,
} from "./core-uiset-record";

export async function loadCoreAssetWorkspace(
  input: GetAssetRecordInput,
): Promise<AssetWorkspaceData> {
  const assetId = persistedAssetId(input.assetId);
  const detail = await coreAssetApi.detail(assetId);
  const [project, recordsResponse] = await Promise.all([
    projectApi.detail(input.projectId),
    coreAssetApi.records(assetId),
  ]);
  const workspaceInput = {
    projectId: input.projectId,
    projectName: project.name,
    records: recordsResponse.records,
  };

  switch (detail.type) {
    case "character":
    case "object":
      return toCoreSpriteAssetWorkspace({ ...workspaceInput, detail });
    case "tileSet":
      return toCoreTilesetAssetWorkspace({ ...workspaceInput, detail });
    case "scenery":
      return toCoreSceneryAssetWorkspace({ ...workspaceInput, detail });
    case "uiset":
      return toCoreUISetAssetWorkspace({ ...workspaceInput, detail });
    case "audio":
      throw new Error("Audio assets do not have editable records.");
  }
}

export async function saveCoreAssetRevision(
  input: SaveAssetRecordInput,
): Promise<AssetRecordSaveResult> {
  const assetId = persistedAssetId(input.assetId);
  const expectedVersion = parseVersion(input.version);
  const saved = await coreAssetApi.record({
    assetId,
    ...(expectedVersion ? { expectedVersion } : {}),
    ...(input.description ? { description: input.description } : {}),
    content: toCoreAssetContent(input.record),
  });
  const records = await coreAssetApi.records(assetId);

  return {
    projectId: input.projectId,
    assetId: input.assetId,
    version: `v${saved.version}`,
    history: toCoreAssetHistory(records.records, saved.version),
    record: structuredClone(input.record),
  };
}

function toCoreAssetContent(record: AssetRecord) {
  switch (record.mode) {
    case "character":
    case "object":
      return toCoreSpriteAssetContent(record);
    case "tileset":
      return toCoreTilesetAssetContent(record);
    case "scenery":
      return toCoreSceneryAssetContent(record);
    case "uiset":
      return toCoreUISetAssetContent(record);
  }
}

function persistedAssetId(value: string) {
  const assetId = Number(value);
  if (!Number.isSafeInteger(assetId) || assetId <= 0) {
    throw new Error("Asset editor requires a persisted Core API asset.");
  }
  return assetId;
}

function parseVersion(version: string | undefined) {
  const value = Number(version?.replace(/^v/, ""));
  return Number.isSafeInteger(value) && value > 0 ? value : undefined;
}
