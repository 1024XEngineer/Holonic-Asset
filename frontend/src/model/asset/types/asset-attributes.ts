import type { AssetContentKind } from "../content/types";
import type { Perspective } from "@/model/project";

export type AssetAttributes<K extends AssetContentKind = AssetContentKind> = {
  id: string;
  projectId: string;
  kind: K;
  name: string;
  description: string;
  tags: string[];
  perspective: Perspective;
  dimensions: unknown;
};
