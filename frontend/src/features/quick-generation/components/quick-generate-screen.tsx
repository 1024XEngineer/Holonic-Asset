import { Download, Plus, RefreshCw, Sparkles, Trash2 } from "lucide-react";

import { useQuickGeneration } from "../state/use-quick-generation";
import { quickGenerationSizes } from "../domain";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { getQuickAssetPreviewClassName } from "./quick-generation.constants";

export function QuickGenerateScreen() {
  const { model, actions } = useQuickGeneration();
  const { assets, currentAsset, currentAssetId, draft, status } = model;
  const {
    chooseReference,
    clearReference,
    deleteCurrentAsset,
    generate,
    reload,
    selectAsset,
    startNewAsset,
    updateDraft,
  } = actions;
  const {
    actionError,
    isDeleting,
    isGenerating,
    isLoading,
    isMutating,
    loadError,
  } = status;

  return (
    <main className="relative flex min-h-[calc(100vh-3.5rem)] flex-1 flex-col bg-muted/30">
      <header className="border-b bg-background px-5 py-4">
        <div className="mx-auto max-w-[100rem]">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-semibold tracking-tight">
                Quick Generating
              </h1>
              <Badge variant="secondary">{assets.length} assets</Badge>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              Create and update standalone assets without project setup.
            </p>
          </div>
        </div>
      </header>

      <div className="mx-auto grid w-full max-w-[100rem] flex-1 xl:h-[calc(100vh-8.5rem)] xl:grid-cols-[16rem_minmax(0,1fr)_21rem] xl:overflow-hidden">
        <aside className="border-b bg-background p-4 xl:border-r xl:border-b-0">
          <Button
            className="w-full"
            variant="outline"
            onClick={startNewAsset}
            disabled={isLoading || isMutating}
          >
            <Plus data-icon="inline-start" />
            New asset
          </Button>
          <div className="mt-5 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-[.14em] text-muted-foreground">
              Assets
            </h2>
            <span className="text-xs text-muted-foreground">
              {assets.length}
            </span>
          </div>
          <div className="mt-2 grid grid-cols-2 gap-2">
            {isLoading ? (
              <p
                className="col-span-2 px-2 py-6 text-center text-xs text-muted-foreground"
                role="status"
              >
                Loading generated assets…
              </p>
            ) : loadError ? (
              <div
                className="col-span-2 space-y-3 px-2 py-6 text-center"
                role="alert"
              >
                <p className="text-xs leading-5 text-destructive">
                  {loadError.message || "Unable to load generated assets."}
                </p>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => void reload()}
                >
                  Try again
                </Button>
              </div>
            ) : (
              assets.map((asset, index) => {
                const active = asset.id === currentAssetId;
                const previewClassName = asset.previewUrl
                  ? "bg-cover bg-center"
                  : getQuickAssetPreviewClassName(asset.id);
                return (
                  <button
                    key={asset.id}
                    type="button"
                    onClick={() => selectAsset(asset)}
                    disabled={isMutating}
                    aria-label={`Generated image ${index + 1}`}
                    className={`aspect-square overflow-hidden rounded-xl border-2 p-1 transition-colors focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50 ${
                      active
                        ? "border-foreground"
                        : "border-transparent hover:border-border"
                    }`}
                  >
                    <span
                      className={`block size-full rounded-lg outline -outline-offset-1 outline-black/10 ${previewClassName}`}
                      style={
                        asset.previewUrl
                          ? { backgroundImage: `url("${asset.previewUrl}")` }
                          : undefined
                      }
                    />
                  </button>
                );
              })
            )}
            {!isLoading && !loadError && assets.length === 0 ? (
              <p className="px-2 py-6 text-center text-xs leading-5 text-muted-foreground">
                Your generated assets will appear here.
              </p>
            ) : null}
          </div>
        </aside>

        <section className="flex min-h-0 min-w-0 flex-col p-5 sm:p-6">
          <div className="mb-4 flex min-h-9 items-center justify-between gap-3">
            {currentAsset ? (
              <Badge variant="secondary">{currentAsset.size}</Badge>
            ) : (
              <span className="font-semibold">New standalone asset</span>
            )}
            {currentAsset ? (
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={deleteCurrentAsset}
                disabled={isMutating}
              >
                {isDeleting ? (
                  <RefreshCw className="animate-spin" />
                ) : (
                  <Trash2 />
                )}
                <span className="sr-only">Delete asset</span>
              </Button>
            ) : null}
          </div>

          <Card className="mx-auto flex aspect-square min-h-72 w-full max-w-[34rem] flex-1 overflow-hidden py-0 xl:min-h-0 xl:flex-none">
            <div className="relative grid min-h-72 flex-1 place-items-center overflow-hidden bg-[linear-gradient(45deg,var(--muted)_25%,transparent_25%),linear-gradient(-45deg,var(--muted)_25%,transparent_25%),linear-gradient(45deg,transparent_75%,var(--muted)_75%),linear-gradient(-45deg,transparent_75%,var(--muted)_75%)] bg-[length:24px_24px] bg-[position:0_0,0_12px,12px_-12px,-12px_0] xl:min-h-0">
              {currentAsset ? (
                <div
                  className={`relative grid aspect-square w-[min(60%,22rem)] place-items-center overflow-hidden rounded-2xl shadow-sm outline -outline-offset-1 outline-black/10 ${
                    currentAsset.previewUrl
                      ? "bg-cover bg-center"
                      : getQuickAssetPreviewClassName(currentAsset.id)
                  }`}
                  style={
                    currentAsset.previewUrl
                      ? {
                          backgroundImage: `url("${currentAsset.previewUrl}")`,
                        }
                      : undefined
                  }
                >
                  <div className="absolute inset-0 bg-[linear-gradient(transparent_55%,rgba(0,0,0,.68))]" />
                  <div className="absolute bottom-0 z-10 w-full p-6 text-white">
                    <p className="mt-2 line-clamp-3 text-lg font-medium">
                      {currentAsset.prompt}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="max-w-sm px-6 text-center">
                  <span className="mx-auto grid size-12 place-items-center rounded-xl border bg-background">
                    <Sparkles className="size-5 text-muted-foreground" />
                  </span>
                  <h2 className="mt-4 font-semibold">
                    {assets.length
                      ? "Create another asset"
                      : "Start with a description"}
                  </h2>
                  <p className="mt-2 text-sm leading-6 text-muted-foreground">
                    Each asset keeps its description, reference image, and
                    latest generated result.
                  </p>
                </div>
              )}
            </div>
          </Card>

          {currentAsset ? (
            <div className="mt-4 flex items-center justify-between gap-3 rounded-xl border bg-background p-4">
              <div>
                <p className="text-sm font-medium">Current asset</p>
                <p className="mt-1 line-clamp-1 text-xs text-muted-foreground">
                  {currentAsset.prompt}
                </p>
              </div>
              <Button>
                <Download data-icon="inline-start" /> Export
              </Button>
            </div>
          ) : null}
        </section>

        <aside className="border-t bg-background p-5 xl:border-t-0 xl:border-l xl:p-6">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[.14em] text-muted-foreground">
              {currentAsset ? "Current asset" : "New asset"}
            </p>
            <h2 className="mt-2 text-lg font-semibold">
              {currentAsset ? "Update the asset" : "Describe the asset"}
            </h2>
            <p className="mt-1 text-sm leading-6 text-muted-foreground">
              {currentAsset
                ? "Generating again updates the current result."
                : "The first description also names the asset."}
            </p>
          </div>

          <label className="mt-6 grid gap-2 text-sm font-medium">
            Size
            <select
              className="h-9 rounded-lg border bg-transparent px-3 text-sm outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
              value={draft.size}
              onChange={(event) => updateDraft({ size: event.target.value })}
            >
              {quickGenerationSizes.map((option) => (
                <option key={option}>{option}</option>
              ))}
            </select>
          </label>

          <div className="mt-5 overflow-hidden rounded-2xl border bg-background shadow-sm focus-within:ring-3 focus-within:ring-ring/50">
            <ImageDropzone
              className="m-3 min-h-20"
              previewUrl={draft.reference || undefined}
              onSelect={chooseReference}
              onClear={clearReference}
            />
            <textarea
              autoFocus
              className="min-h-44 w-full resize-none bg-transparent px-4 py-3 text-sm leading-6 outline-none placeholder:text-muted-foreground"
              placeholder={
                currentAsset
                  ? "Describe how you want to update this asset, you can upload or drag an image here for reference"
                  : "Describe the asset you want to create, you can upload or drag an image here for reference"
              }
              value={draft.prompt}
              onChange={(event) => updateDraft({ prompt: event.target.value })}
            />
            <div className="flex items-center justify-between gap-3 p-3 pt-0">
              <span className="text-xs leading-4 text-muted-foreground">
                PNG, JPEG, or WebP
              </span>
              <Button
                disabled={!draft.prompt.trim() || isLoading || isMutating}
                onClick={generate}
              >
                {isGenerating ? (
                  <RefreshCw className="animate-spin" />
                ) : (
                  <Sparkles />
                )}
                {currentAsset ? "Update asset" : "Generate asset"}
              </Button>
            </div>
          </div>
          {actionError ? (
            <p className="mt-3 text-sm text-destructive" role="alert">
              {actionError.message || "Unable to update this asset."}
            </p>
          ) : null}
        </aside>
      </div>
    </main>
  );
}
