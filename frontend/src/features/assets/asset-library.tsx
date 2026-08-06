import type { ProjectSummary } from "@/model/project";

import { AssetLibraryWorkspace } from "./asset-library-workspace";
import { useAssetLibraryController } from "./state/use-asset-library-controller";

export function AssetLibrary({ project }: { project?: ProjectSummary }) {
  const library = useAssetLibraryController({ project });

  return <AssetLibraryWorkspace library={library} />;
}
