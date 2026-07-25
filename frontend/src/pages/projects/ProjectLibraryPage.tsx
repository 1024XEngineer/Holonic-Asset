import { AssetLibraryWorkspace } from "@/features/AssetLibrary";
import { CreateAssetToolbar, GenerationQueue } from "@/features/Generation";
import { ProjectSidebar } from "@/features/Project";
import { AppHeader } from "@/components/layouts/AppHeader";
import { ProjectChrome } from "@/components/layouts/ProjectChrome";
import { creatableAssetKinds } from "@/domain/asset";
import { useProjectLibrary } from "./useProjectLibrary";

export function ProjectLibraryPage() {
  const library = useProjectLibrary();

  return (
    <div className="min-h-screen bg-background text-foreground">
      <AppHeader />
      <ProjectChrome
        sidebar={
          <ProjectSidebar
            isProjectRoute
            projects={library.project.items}
            selectedProjectId={library.project.selectedId}
            onCreateProject={() => void library.project.create()}
            onDeleteProject={(projectId) =>
              void library.project.remove(projectId)
            }
            onSelectProject={library.project.select}
            onUpdateProject={library.project.update}
          />
        }
      >
        <AssetLibraryWorkspace
          assetGroups={library.assetLibrary.groups}
          project={library.project.current}
          query={library.assetLibrary.query}
          generationQueue={<GenerationQueue runs={library.generation.runs} />}
          creationControl={
            library.project.current ? (
              <CreateAssetToolbar
                assetKinds={creatableAssetKinds}
                project={library.project.current}
                onCreate={library.assetLibrary.createAsset}
              />
            ) : null
          }
          onCopyAsset={library.assetLibrary.copyAsset}
          onDeleteAsset={library.assetLibrary.deleteAsset}
          onOpenAsset={library.assetLibrary.openAsset}
          onQueryChange={library.assetLibrary.changeQuery}
        />
      </ProjectChrome>
    </div>
  );
}
