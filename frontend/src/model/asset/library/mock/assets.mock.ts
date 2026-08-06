import type { AssetKind, AssetMetadataUpdate, ProjectAsset } from "../../types";
import { createAssetLibraryCollection } from "../asset-library-collection";
import type { AssetGroupsByProject } from "../types";
import { assetGroupsByProject as seededAssetGroups } from "./assets.seed";

let assetGroupsByProject = createAssetState();

function createAssetState(): AssetGroupsByProject {
  return structuredClone(seededAssetGroups);
}

export async function listMockAssetGroups(projectId: string) {
  return structuredClone(assetGroupsByProject[projectId] ?? []);
}

export async function addMockAsset(
  projectId: string,
  kind: AssetKind,
  asset: ProjectAsset,
) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const collection = createAssetLibraryCollection(groups, { projectId });
  const updatedGroups = collection.append(kind, structuredClone(asset));

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: updatedGroups,
  };
  return structuredClone(updatedGroups);
}

export async function copyMockAsset(projectId: string, assetId: string) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const copyId = `${assetId}-copy-${crypto.randomUUID()}`;
  const collection = createAssetLibraryCollection(groups, { projectId });
  const updatedGroups = collection.insertAfter(assetId, (asset) => ({
    ...asset,
    id: copyId,
    name: `${asset.name} Copy`,
    history: asset.history.map((entry) => ({
      ...entry,
      id: `${copyId}-history-${entry.version}`,
    })),
    animations: asset.animations.map((animation) => ({
      ...animation,
      id: `${copyId}-animation-${animation.id}`,
    })),
  }));

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: updatedGroups,
  };
  return structuredClone(updatedGroups);
}

export async function deleteMockAsset(projectId: string, assetId: string) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const collection = createAssetLibraryCollection(groups, { projectId });
  const updatedGroups = collection.remove(assetId);

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: updatedGroups,
  };
  return structuredClone(updatedGroups);
}

export async function updateMockAsset(
  projectId: string,
  assetId: string,
  metadata: AssetMetadataUpdate,
) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const collection = createAssetLibraryCollection(groups, { projectId });
  const updatedGroups = collection.update(assetId, (asset) => ({
    ...asset,
    ...structuredClone(metadata),
  }));

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: updatedGroups,
  };
  return structuredClone(updatedGroups);
}

export async function saveMockAssetRevision<Payload>(
  projectId: string,
  assetId: string,
  description: string,
  payload: Payload,
) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const savedAt = new Date();
  const collection = createAssetLibraryCollection(groups, { projectId });
  const updatedGroups = collection.update(assetId, (asset) => {
    const version = nextAssetVersion(asset.version);
    const record = {
      id: `record-${asset.id}-${crypto.randomUUID()}`,
      version,
      description: description.trim() || asset.description,
      status: "ready" as const,
      isCurrent: true,
      content: structuredClone(payload),
      savedAt: savedAt.toISOString(),
    };

    return {
      ...asset,
      version,
      description: record.description,
      history: [
        record,
        ...asset.history.map((entry) => ({ ...entry, isCurrent: false })),
      ],
    };
  });

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: updatedGroups,
  };
  return structuredClone(updatedGroups);
}

export function deleteMockProjectAssets(projectId: string) {
  const { [projectId]: _, ...remainingAssets } = assetGroupsByProject;
  assetGroupsByProject = remainingAssets;
}

export function resetMockAssets() {
  assetGroupsByProject = createAssetState();
}

function nextAssetVersion(version: string) {
  const current = Number(version.slice(1));
  return Number.isInteger(current) && current >= 0 ? `v${current + 1}` : "v1";
}
