import { assetGroupsByProject, projectSummaries } from "./project.seed";
import type { AssetGroupsByProject } from "@/types/asset-library";
import type { ProjectAsset } from "@/types/asset";
import type { AssetKind } from "@/types/asset-kind";
import type { GenerationRun } from "@/types/generation";
import type { ProjectSummary } from "@/types/project";
import {
  isRecordContentForAssetKind,
  type RecordContent,
} from "@/types/record";

type MockWorkspaceState = {
  projects: ProjectSummary[];
  assetGroupsByProject: AssetGroupsByProject;
  generationRuns: GenerationRun[];
};

let workspace = createMockWorkspaceState();

function createMockWorkspaceState(): MockWorkspaceState {
  return {
    projects: structuredClone(projectSummaries),
    assetGroupsByProject: structuredClone(assetGroupsByProject),
    generationRuns: [],
  };
}

function titleForKind(kind: AssetKind) {
  return kind === "ui" ? "UI" : `${kind[0].toUpperCase()}${kind.slice(1)}`;
}

export async function listMockProjects() {
  return structuredClone(workspace.projects);
}

export async function createMockProject(project: ProjectSummary) {
  workspace.projects = [...workspace.projects, structuredClone(project)];
  return structuredClone(project);
}

export async function updateMockProject(project: ProjectSummary) {
  workspace.projects = workspace.projects.map((item) =>
    item.id === project.id ? structuredClone(project) : item,
  );
  return structuredClone(project);
}

export async function deleteMockProject(projectId: string) {
  workspace.projects = workspace.projects.filter(
    (project) => project.id !== projectId,
  );
  const { [projectId]: _, ...remainingAssets } = workspace.assetGroupsByProject;
  workspace.assetGroupsByProject = remainingAssets;
  workspace.generationRuns = workspace.generationRuns.filter(
    (run) => run.projectId !== projectId,
  );
}

export function hasMockProject(projectId: string) {
  return workspace.projects.some((project) => project.id === projectId);
}

export async function listMockAssetGroups(projectId: string) {
  return structuredClone(workspace.assetGroupsByProject[projectId] ?? []);
}

export async function addMockAsset(
  projectId: string,
  kind: AssetKind,
  asset: ProjectAsset,
) {
  const groups = workspace.assetGroupsByProject[projectId] ?? [];
  const group = groups.find((item) => item.kind === kind);

  workspace.assetGroupsByProject = {
    ...workspace.assetGroupsByProject,
    [projectId]: group
      ? groups.map((item) =>
          item.kind === kind
            ? { ...item, assets: [...item.assets, structuredClone(asset)] }
            : item,
        )
      : [
          ...groups,
          {
            kind,
            title: titleForKind(kind),
            accentClassName: "bg-slate-500",
            assets: [structuredClone(asset)],
          },
        ],
  };
  return structuredClone(workspace.assetGroupsByProject[projectId]);
}

export async function copyMockAsset(projectId: string, assetId: string) {
  const groups = workspace.assetGroupsByProject[projectId] ?? [];
  const copyId = `${assetId}-copy-${crypto.randomUUID()}`;

  workspace.assetGroupsByProject = {
    ...workspace.assetGroupsByProject,
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
  return structuredClone(workspace.assetGroupsByProject[projectId]);
}

export async function deleteMockAsset(projectId: string, assetId: string) {
  workspace.assetGroupsByProject = {
    ...workspace.assetGroupsByProject,
    [projectId]: (workspace.assetGroupsByProject[projectId] ?? []).map(
      (group) => ({
        ...group,
        assets: group.assets.filter((asset) => asset.id !== assetId),
      }),
    ),
  };
  return structuredClone(workspace.assetGroupsByProject[projectId]);
}

export async function saveMockAssetRevision(
  projectId: string,
  assetId: string,
  content: RecordContent,
) {
  const groups = workspace.assetGroupsByProject[projectId] ?? [];
  const assetGroup = groups.find((group) =>
    group.assets.some((asset) => asset.id === assetId),
  );
  if (assetGroup && !isRecordContentForAssetKind(assetGroup.kind, content)) {
    throw new Error("Record content does not match the asset kind.");
  }
  const savedAt = new Date();

  workspace.assetGroupsByProject = {
    ...workspace.assetGroupsByProject,
    [projectId]: groups.map((group) => ({
      ...group,
      assets: group.assets.map((asset) => {
        if (asset.id !== assetId) return asset;

        const version = nextAssetVersion(asset.version);
        const record = {
          id: `record-${asset.id}-${crypto.randomUUID()}`,
          version,
          description: content.prompt.trim() || asset.description,
          status: "ready" as const,
          isCurrent: true,
          content: structuredClone(content),
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
  return structuredClone(workspace.assetGroupsByProject[projectId]);
}

export async function listMockGenerationRuns(projectId: string) {
  return structuredClone(
    workspace.generationRuns.filter((run) => run.projectId === projectId),
  );
}

export function createMockGenerationRun(
  run: Omit<GenerationRun, "id" | "status">,
) {
  const createdRun: GenerationRun = {
    ...structuredClone(run),
    id: crypto.randomUUID(),
    status: "queued",
  };
  workspace.generationRuns = [...workspace.generationRuns, createdRun];
  return structuredClone(createdRun);
}

export function updateMockGenerationRun(run: GenerationRun) {
  workspace.generationRuns = workspace.generationRuns.map((item) =>
    item.id === run.id ? structuredClone(run) : item,
  );
  return structuredClone(run);
}

export function removeMockGenerationRun(runId: string) {
  workspace.generationRuns = workspace.generationRuns.filter(
    (run) => run.id !== runId,
  );
}

export function resetMockWorkspace() {
  workspace = createMockWorkspaceState();
}

function nextAssetVersion(version: string) {
  const current = Number(version.slice(1));
  return Number.isInteger(current) && current >= 0 ? `v${current + 1}` : "v1";
}
