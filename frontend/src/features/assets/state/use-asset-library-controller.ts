import type { AssetKind, ProjectAsset } from "@/model/asset";
import type { CreationRequest, GenerationRun } from "@/model/generation";
import type { ProjectSummary } from "@/model/project";

type FilteredAsset = ProjectAsset & {
  kind: AssetKind;
  accentClassName: string;
  kindLabel: string;
};

export type AssetLibraryController = {
  project?: ProjectSummary;
  query: string;
  selectedKinds: AssetKind[];
  filteredAssets: FilteredAsset[];
  generationRuns: GenerationRun[];
  createAsset: (request: CreationRequest) => void;
  copyAsset: (assetId: string) => void;
  deleteAsset: (assetId: string) => void;
  openAsset: (assetId: string) => void;
  setQuery: (query: string) => void;
  setSelectedKinds: (kinds: AssetKind[]) => void;
};
