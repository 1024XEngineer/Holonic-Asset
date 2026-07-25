import { Button } from "@/components/ui/button";

import type { ProjectSummary } from "@/domain/project";
import { AssetTypeIcon } from "@/components/ui/custom/AssetTypeIcon";
import { getAssetTypeConfig } from "@/components/ui/custom/asset-type-config";
import type { CreatableAssetKind } from "@/domain/asset";
import type { CreationRequest } from "@/domain/generation";
import { CreateAssetDialog } from "./create-asset-dialog";

export function CreateAssetToolbar({
  assetKinds,
  onCreate,
  project,
}: {
  assetKinds: CreatableAssetKind[];
  onCreate: (request: CreationRequest) => void;
  project: ProjectSummary;
}) {
  return (
    <CreateAssetDialog project={project} onCreate={onCreate}>
      {(openDialog) => (
        <div className="grid grid-cols-1 rounded-2xl border bg-background p-1 shadow-sm sm:grid-cols-2 lg:grid-cols-5">
          {assetKinds.map((kind) => (
            <Button
              key={kind}
              type="button"
              variant="ghost"
              className="h-10 justify-start rounded-xl px-3 text-sm sm:justify-center lg:px-4"
              onClick={() => openDialog(kind)}
            >
              <AssetTypeIcon kind={kind} className="size-5" />
              Create {getAssetTypeConfig(kind).label}
            </Button>
          ))}
        </div>
      )}
    </CreateAssetDialog>
  );
}
