import { projectApi } from "../../project";
import { coreAssetApi } from "../library/core-asset.api";
import type {
  AssetRecord,
  AssetRecordApi,
  AssetRecordRollbackResult,
  AssetRecordSaveResult,
  AssetWorkspaceData,
  GetAssetRecordInput,
  RollbackAssetRecordInput,
  SaveAssetRecordInput,
} from "./types";
import { toAssetHistory } from "./record-history.mapper";
import {
  toBackendSceneryContent,
  toSceneryAssetContent,
} from "../content/scenery-content.mapper";
import {
  toBackendSpriteContent,
  toSpriteAssetContent,
} from "../content/sprite-content.mapper";
import {
  toBackendTilesetContent,
  toTilesetAssetContent,
} from "../content/tileset-content.mapper";
import {
  toBackendUISetContent,
  toUISetAssetContent,
} from "../content/uiset-content.mapper";

export async function loadAssetSnapshot(
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
      return toSpriteAssetContent({ ...workspaceInput, detail });
    case "tileSet":
      return toTilesetAssetContent({ ...workspaceInput, detail });
    case "scenery":
      return toSceneryAssetContent({ ...workspaceInput, detail });
    case "uiset":
      return toUISetAssetContent({ ...workspaceInput, detail });
    case "audio":
      throw new Error("Audio assets do not have editable records.");
  }
}

export async function saveAssetRevision(
  input: SaveAssetRecordInput,
): Promise<AssetRecordSaveResult> {
  const assetId = persistedAssetId(input.assetId);
  const expectedVersion = parseVersion(input.version);
  const saved = await coreAssetApi.record({
    assetId,
    ...(expectedVersion ? { expectedVersion } : {}),
    ...(input.description ? { description: input.description } : {}),
    content: toBackendContent(input.record),
  });
  const records = await coreAssetApi.records(assetId);

  return {
    projectId: input.projectId,
    assetId: input.assetId,
    version: `v${saved.version}`,
    history: toAssetHistory(records.records, saved.version),
    record: structuredClone(input.record),
  };
}

export async function rollbackAssetRecord(
  input: RollbackAssetRecordInput,
): Promise<AssetRecordRollbackResult> {
  const result = await coreAssetApi.rollback({
    assetId: persistedAssetId(input.assetId),
    version: parseVersion(input.version) ?? 0,
  });
  if (!result.version || !result.contentId) {
    throw new Error("Asset rollback returned an invalid record.");
  }
  return {
    projectId: input.projectId,
    assetId: input.assetId,
    version: `v${result.version}`,
    contentId: String(result.contentId),
  };
}

function toBackendContent(record: AssetRecord) {
  switch (record.mode) {
    case "character":
    case "object":
      return toBackendSpriteContent(record);
    case "tileset":
      return toBackendTilesetContent(record);
    case "scenery":
      return toBackendSceneryContent(record);
    case "uiset":
      return toBackendUISetContent(record);
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

export const assetRecordApi: AssetRecordApi = {
  load: loadAssetSnapshot,
  saveRevision: saveAssetRevision,
  rollback: rollbackAssetRecord,
};

/** @deprecated Use assetRecordApi. */
export const assetWorkspaceApi = assetRecordApi;

export type {
  AssetRecordApi,
  AssetRecordSaveResult,
  GetAssetRecordInput,
  SaveAssetRecordInput,
  RollbackAssetRecordInput,
  AssetRecordRollbackResult,
} from "./types";
