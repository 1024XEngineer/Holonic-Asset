import type { ProjectSummary } from "@/model/project";

import { AssetLibraryWorkspace } from "./asset-library-workspace";
import { useAssetLibraryController } from "./state/use-asset-library-controller";

export function AssetLibrary({
  isProjectLoading = false,
  project,
  projectError,
  retryProject,
}: {
  isProjectLoading?: boolean;
  project?: ProjectSummary;
  projectError?: Error;
  retryProject: () => void;
}) {
  const library = useAssetLibraryController({ project });

  return (
    <AssetLibraryWorkspace
      isProjectLoading={isProjectLoading}
      library={library}
      projectError={projectError}
      retryProject={retryProject}
    />
  );
}
