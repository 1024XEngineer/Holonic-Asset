import type { AssetKind, ProjectAsset } from "@/model/asset";

export type AssetLibraryItem = ProjectAsset & {
  kind: AssetKind;
  accentClassName: string;
  kindLabel: string;
};
