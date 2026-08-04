import type { AssetKind } from "@/model/asset";

export type Asset = {
  id: string;
  name: string;
  kind: AssetKind;
  version: string;
  size: string;
  description: string;
  tags: string[];
  accent: string;
};
