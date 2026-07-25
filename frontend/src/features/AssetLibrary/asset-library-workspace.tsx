import { ScrollArea } from "@/components/ui/scroll-area";

import { useAssetLibrary } from "./useAssetLibrary";
import type { ProjectSummary } from "@/domain/project";
import type { AssetGroup } from "@/domain/asset";
import { AssetCard } from "./asset-card";
import { AssetLibraryToolbar } from "./asset-library-toolbar";

type AssetLibraryWorkspaceProps = {
  assetGroups: AssetGroup[];
  creationControl: React.ReactNode;
  generationQueue: React.ReactNode;
  onCopyAsset: (assetId: string) => void;
  onDeleteAsset: (assetId: string) => void;
  onOpenAsset: (assetId: string) => void;
  onQueryChange: (query: string) => void;
  project?: ProjectSummary;
  query: string;
};

export function AssetLibraryWorkspace({
  assetGroups,
  creationControl,
  generationQueue,
  onCopyAsset,
  onDeleteAsset,
  onOpenAsset,
  onQueryChange,
  project,
  query,
}: AssetLibraryWorkspaceProps) {
  const { filteredAssets, selectedKinds, setSelectedKinds } = useAssetLibrary(
    assetGroups,
    query,
  );

  if (!project) {
    return (
      <div className="flex h-full items-center justify-center px-6 text-center">
        <p className="max-w-md text-sm leading-6 text-muted-foreground">
          Please create a project or select an existing project from the project
          list on the left.
        </p>
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="mx-auto w-full max-w-[96rem] px-5 py-7 sm:px-8 sm:py-9">
        <div className="pb-10 sm:pb-12">
          <AssetLibraryToolbar
            query={query}
            selectedKinds={selectedKinds}
            creationControl={creationControl}
            onQueryChange={onQueryChange}
            onSelectedKindsChange={setSelectedKinds}
          />
        </div>

        {generationQueue}

        {filteredAssets.length > 0 ? (
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-5">
            {filteredAssets.map((asset) => (
              <AssetCard
                key={asset.id}
                asset={asset}
                accentClassName={asset.accentClassName}
                kind={asset.kind}
                kindLabel={asset.kindLabel}
                onOpen={() => onOpenAsset(asset.id)}
                onCopy={() => onCopyAsset(asset.id)}
                onDelete={() => onDeleteAsset(asset.id)}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-2xl border border-dashed bg-card px-6 py-20 text-center">
            <p className="text-sm font-medium">No assets found</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Try another search or asset type.
            </p>
          </div>
        )}
      </div>
    </ScrollArea>
  );
}
