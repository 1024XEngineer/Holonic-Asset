import type { CreatableAssetKind } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import type { ProjectSummary } from "@/model/project";

export function CreateAssetToolbar(_props: {
  assetKinds: CreatableAssetKind[];
  onCreate: (request: CreationRequest) => void;
  project: ProjectSummary;
}) {
  return null;
}
