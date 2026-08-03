import type { CreatableAssetKind } from "@/features/assets/types";
import type { ProjectSummary } from "@/features/project";

import type { CreationRequest } from "./types";

export function CreateAssetToolbar(_props: {
  assetKinds: CreatableAssetKind[];
  onCreate: (request: CreationRequest) => void;
  project: ProjectSummary;
}) {
  return null;
}
