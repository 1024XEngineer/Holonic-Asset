import { Copy, LoaderCircle, Pencil, Trash2 } from "lucide-react";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { AssetKindIcon, getAssetKindConfig } from "@/components/asset-kind";
import type { AssetLibraryItem } from "@/model/asset";

import { AssetPreview } from "./asset-preview";

export function AssetCard({
  asset,
  isCopying,
  isDeleting,
  onCopy,
  onDelete,
  onEdit,
  onOpenEditor,
  projectId,
}: {
  asset: AssetLibraryItem;
  isCopying: boolean;
  isDeleting: boolean;
  onCopy: () => void;
  onDelete: () => void;
  onEdit: () => void;
  onOpenEditor?: () => void;
  projectId?: string;
}) {
  const kindConfig = getAssetKindConfig(asset.kind);

  return (
    <Card
      className="group relative gap-0 overflow-hidden rounded-lg py-0 shadow-xs transition-[box-shadow,transform] hover:-translate-y-0.5 hover:shadow-md has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/40"
      size="sm"
    >
      {onOpenEditor ? (
        <Tooltip>
          <TooltipTrigger
            render={
              <div
                role="link"
                tabIndex={0}
                aria-label={`Open ${asset.name} editor`}
                className="absolute inset-0 z-10 cursor-pointer focus-visible:outline-none"
                onClick={onOpenEditor}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onOpenEditor();
                  }
                }}
              />
            }
          />
          <TooltipContent side="bottom">Open asset editor</TooltipContent>
        </Tooltip>
      ) : null}
      <div className="min-w-0">
        <AssetPreview asset={asset} projectId={projectId} />
        <div className="space-y-3 p-3.5">
          <div className="flex min-w-0 items-start justify-between gap-3">
            <div className="min-w-0">
              <h3 className="truncate text-sm font-semibold">{asset.name}</h3>
              <p className="mt-1 line-clamp-2 min-h-9 text-xs leading-[1.125rem] text-muted-foreground">
                {asset.description}
              </p>
              {asset.tags.length > 0 ? (
                <div className="mt-2 flex flex-wrap gap-1" aria-label="Tags">
                  {asset.tags.map((tag) => (
                    <Badge
                      key={tag}
                      variant="secondary"
                      className="h-4 rounded-sm px-1 text-[10px] font-medium leading-none"
                    >
                      {tag}
                    </Badge>
                  ))}
                </div>
              ) : null}
            </div>
            <Badge variant="outline" className="rounded-md">
              {asset.version}
            </Badge>
          </div>
          <div className="flex items-center justify-between gap-3 border-t pt-3 text-xs text-muted-foreground">
            <span className="flex min-w-0 items-center gap-1.5">
              <AssetKindIcon kind={asset.kind} className="size-3.5" />
              <span className="truncate">{kindConfig.label}</span>
            </span>
            <span className="shrink-0">{asset.canvasSize}</span>
          </div>
        </div>
      </div>

      <div className="absolute left-2 top-2 z-30 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                aria-label={`Copy ${asset.name}`}
                variant="outline"
                size="icon-sm"
                className="bg-background/90 shadow-xs backdrop-blur-sm"
                disabled={isCopying || isDeleting}
                onClick={onCopy}
              />
            }
          >
            {isCopying ? <LoaderCircle className="animate-spin" /> : <Copy />}
          </TooltipTrigger>
          <TooltipContent>Copy asset</TooltipContent>
        </Tooltip>
      </div>

      <div className="absolute right-2 top-2 z-30 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                aria-label={`Edit ${asset.name} details`}
                variant="outline"
                size="icon-sm"
                className="bg-background/90 shadow-xs backdrop-blur-sm"
                disabled={isCopying || isDeleting}
                onClick={onEdit}
              />
            }
          >
            <Pencil />
          </TooltipTrigger>
          <TooltipContent>Edit asset details</TooltipContent>
        </Tooltip>
        <AlertDialog>
          <Tooltip>
            <TooltipTrigger render={<span className="inline-flex" />}>
              <AlertDialogTrigger
                render={
                  <Button
                    aria-label={`Delete ${asset.name}`}
                    variant="outline"
                    size="icon-sm"
                    className="bg-background/90 text-muted-foreground shadow-xs backdrop-blur-sm hover:bg-destructive/10 hover:text-destructive"
                    disabled={isCopying || isDeleting}
                  />
                }
              >
                <Trash2 />
              </AlertDialogTrigger>
            </TooltipTrigger>
            <TooltipContent>Delete asset</TooltipContent>
          </Tooltip>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {asset.name}?</AlertDialogTitle>
              <AlertDialogDescription>
                This removes the asset and its saved records from this project.
                This action cannot be undone.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction variant="destructive" onClick={onDelete}>
                Delete asset
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </Card>
  );
}
