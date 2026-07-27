import type {
  AssetGroupsByProject,
  AssetKind,
  ProjectAsset,
} from "../../domain";
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
  const group = groups.find((item) => item.kind === kind);

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: group
      ? groups.map((item) =>
          item.kind === kind
            ? { ...item, assets: [...item.assets, structuredClone(asset)] }
            : item,
        )
      : [...groups, { kind, assets: [structuredClone(asset)] }],
  };
  return structuredClone(assetGroupsByProject[projectId]);
}

export async function copyMockAsset(projectId: string, assetId: string) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const copyId = `${assetId}-copy-${crypto.randomUUID()}`;

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: groups.map((group) => {
      const assetIndex = group.assets.findIndex(
        (asset) => asset.id === assetId,
      );
      if (assetIndex < 0) return group;

      const asset = group.assets[assetIndex];
      const copiedAsset = {
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
      };

      return {
        ...group,
        assets: [
          ...group.assets.slice(0, assetIndex + 1),
          copiedAsset,
          ...group.assets.slice(assetIndex + 1),
        ],
      };
    }),
  };
  return structuredClone(assetGroupsByProject[projectId]);
}

export async function deleteMockAsset(projectId: string, assetId: string) {
  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: (assetGroupsByProject[projectId] ?? []).map((group) => ({
      ...group,
      assets: group.assets.filter((asset) => asset.id !== assetId),
    })),
  };
  return structuredClone(assetGroupsByProject[projectId]);
}

export async function saveMockAssetRevision<Payload>(
  projectId: string,
  assetId: string,
  description: string,
  payload: Payload,
) {
  const groups = assetGroupsByProject[projectId] ?? [];
  const savedAt = new Date();

  assetGroupsByProject = {
    ...assetGroupsByProject,
    [projectId]: groups.map((group) => ({
      ...group,
      assets: group.assets.map((asset) => {
        if (asset.id !== assetId) return asset;

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
      }),
    })),
  };
  return structuredClone(assetGroupsByProject[projectId]);
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
