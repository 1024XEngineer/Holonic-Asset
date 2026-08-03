import type { CreatableAssetKind } from "@/model/asset/types";
import type { CreationRequest } from "@/model/generation/run/types";
import type { ProjectSummary } from "@/model/project/types";

export function CreateAssetToolbar(_props: {
  assetKinds: CreatableAssetKind[];
  onCreate: (request: CreationRequest) => void;
  project: ProjectSummary;
}) {
  return null;
}
