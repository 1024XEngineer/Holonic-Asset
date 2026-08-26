import { AlertCircle, FolderOpen, SearchX } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { GenerationQueue } from "@/features/generation";

import { AssetCard } from "./asset-card";
import { AssetEditDialog } from "./asset-edit-dialog";
import { AssetLibraryToolbar } from "./asset-library-toolbar";
import type { AssetLibraryController } from "./state/use-asset-library-controller";

export function AssetLibraryWorkspace({
  isProjectLoading = false,
  library,
  projectError,
  retryProject,
}: {
  isProjectLoading?: boolean;
  library: AssetLibraryController;
  projectError?: Error;
  retryProject: () => void;
}) {
  const { t } = useTranslation("assets");
  const project = library.project;
  const navigate = useNavigate();

  if (isProjectLoading) {
    return (
      <ScrollArea className="h-full">
        <div className="mx-auto w-full max-w-[92rem] px-5 py-6 sm:px-7 lg:px-9 lg:py-8">
          <div className="mb-6 h-8 w-48 animate-pulse rounded bg-muted" />
          <AssetLibrarySkeleton />
        </div>
      </ScrollArea>
    );
  }

  if (projectError && !project) {
    return (
      <ScrollArea className="h-full">
        <div className="mx-auto w-full max-w-[92rem] px-5 py-6 sm:px-7 lg:px-9 lg:py-8">
          <AssetLibraryError
            message={projectError.message}
            onRetry={retryProject}
          />
        </div>
      </ScrollArea>
    );
  }

  if (!project) {
    return (
      <div className="grid h-full place-items-center px-6 text-center">
        <div className="max-w-sm">
          <FolderOpen className="mx-auto size-8 text-muted-foreground" />
          <h1 className="mt-4 text-base font-semibold">{t("selectProject")}</h1>
          <p className="mt-1 text-sm leading-6 text-muted-foreground">
            {t("selectProjectDescription")}
          </p>
        </div>
      </div>
    );
  }

  let assetContent: React.ReactNode;
  if (library.isLoading) {
    assetContent = <AssetLibrarySkeleton />;
  } else if (library.error) {
    assetContent = (
      <AssetLibraryError
        message={library.error.message}
        onRetry={library.retry}
      />
    );
  } else if (library.filteredAssets.length > 0) {
    assetContent = (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {library.filteredAssets.map((asset) => (
          <AssetCard
            key={asset.id}
            asset={asset}
            isCopying={library.copyingAssetIds.has(asset.id)}
            isDeleting={library.deletingAssetIds.has(asset.id)}
            onCopy={() => library.copyAsset(asset.id)}
            onDelete={() => library.deleteAsset(asset.id)}
            onEdit={() => library.openAssetEditor(asset.id)}
            onOpenEditor={
              asset.kind !== "audio"
                ? () =>
                    void navigate({
                      to: "/projects/$projectId/assets/$assetId",
                      params: {
                        projectId: project.id,
                        assetId: asset.id,
                      },
                    })
                : undefined
            }
            projectId={project.id}
          />
        ))}
      </div>
    );
  } else {
    assetContent = (
      <AssetLibraryEmptyState
        hasAssets={library.totalAssets > 0}
        onReset={library.clearFilters}
      />
    );
  }

  return (
    <>
      <ScrollArea className="h-full">
        <div className="mx-auto w-full max-w-[92rem] px-5 py-6 sm:px-7 lg:px-9 lg:py-8">
          <AssetLibraryToolbar library={library} />

          <GenerationQueue runs={library.generationRuns} />

          {library.actionError ? (
            <div
              className="mt-5 flex items-start gap-3 border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-destructive"
              role="status"
            >
              <AlertCircle className="mt-0.5 size-4 shrink-0" />
              <p>{library.actionError.message}</p>
            </div>
          ) : null}

          <div className="pt-6">{assetContent}</div>
        </div>
      </ScrollArea>

      <AssetEditDialog
        asset={library.editingAsset}
        availableTags={library.availableTags}
        error={library.updateError}
        isSaving={library.isUpdatingAsset}
        onCreateTag={library.createTag}
        onClose={library.closeAssetEditor}
        onSave={library.updateAsset}
        onUpdateTag={library.updateTag}
        projectId={project.id}
      />
    </>
  );
}

function AssetLibrarySkeleton() {
  const { t } = useTranslation("assets");
  const skeletonIds = [
    "skeleton-1",
    "skeleton-2",
    "skeleton-3",
    "skeleton-4",
    "skeleton-5",
    "skeleton-6",
    "skeleton-7",
    "skeleton-8",
  ];
  return (
    <div
      className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      aria-label={t("loading")}
      role="status"
    >
      {skeletonIds.map((skeletonId) => (
        <div
          key={skeletonId}
          className="overflow-hidden rounded-lg border bg-card"
        >
          <div className="aspect-[4/3] animate-pulse bg-muted" />
          <div className="space-y-3 p-3.5">
            <div className="h-4 w-2/3 animate-pulse rounded bg-muted" />
            <div className="h-3 w-full animate-pulse rounded bg-muted" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-muted" />
          </div>
        </div>
      ))}
    </div>
  );
}

function AssetLibraryError({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  const { t } = useTranslation(["assets", "common"]);
  return (
    <div className="border border-destructive/25 bg-background px-6 py-14 text-center">
      <AlertCircle className="mx-auto size-7 text-destructive" />
      <h2 className="mt-4 text-sm font-semibold">{t("loadError")}</h2>
      <p className="mx-auto mt-1 max-w-md text-sm leading-6 text-muted-foreground">
        {message}
      </p>
      <Button variant="outline" className="mt-5" onClick={onRetry}>
        {t("common:actions.tryAgain")}
      </Button>
    </div>
  );
}

function AssetLibraryEmptyState({
  hasAssets,
  onReset,
}: {
  hasAssets: boolean;
  onReset: () => void;
}) {
  const { t } = useTranslation("assets");
  return (
    <div className="border border-dashed bg-background px-6 py-16 text-center">
      {hasAssets ? (
        <SearchX className="mx-auto size-7 text-muted-foreground" />
      ) : (
        <FolderOpen className="mx-auto size-7 text-muted-foreground" />
      )}
      <h2 className="mt-4 text-sm font-semibold">
        {hasAssets ? t("noMatching") : t("none")}
      </h2>
      <p className="mt-1 text-sm text-muted-foreground">
        {hasAssets ? t("trySearch") : t("empty")}
      </p>
      {hasAssets ? (
        <Button variant="outline" className="mt-5" onClick={onReset}>
          {t("reset")}
        </Button>
      ) : null}
    </div>
  );
}
