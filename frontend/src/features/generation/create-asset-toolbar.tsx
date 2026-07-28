import { Button } from "@/components/ui/button";

import type { ProjectSummary } from "@/model";
import { AssetTypeIcon } from "@/features/assets";
import { getAssetTypeConfig } from "@/features/assets";
import type { CreatableAssetKind } from "@/model";
import type { CreationRequest } from "@/model";
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
        <div className="grid grid-cols-1 rounded-2xl border bg-background p-1 shadow-sm sm:grid-cols-2 lg:grid-cols-6">
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
