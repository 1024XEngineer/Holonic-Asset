import { Copy, Trash2 } from "lucide-react";

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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import type { ProjectAsset } from "@/features/assets/domain";
import type { AssetKind } from "@/features/assets/domain";
import { AssetTypeIcon } from "./asset-type-icon";
import { AssetPreview } from "./asset-preview";

export function AssetCard({
  asset,
  accentClassName,
  kind,
  kindLabel,
  onOpen,
  onCopy,
  onDelete,
}: {
  asset: ProjectAsset;
  accentClassName: string;
  kind: AssetKind;
  kindLabel: string;
  onOpen: () => void;
  onCopy: () => void;
  onDelete: () => void;
}) {
  return (
    <Card
      className="group relative gap-0 overflow-hidden rounded-xl py-0 shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50"
      size="sm"
    >
      <button
        type="button"
        aria-label={`Open ${asset.name}`}
        className="absolute inset-0 z-0 rounded-xl focus-visible:outline-none"
        onClick={onOpen}
      />
      <div className="pointer-events-none relative z-10">
        <AssetPreview
          accentClassName={accentClassName}
          className="aspect-[4/3]"
        />
        <CardHeader className="px-3 pt-3">
          <CardTitle className="truncate text-sm font-semibold">
            {asset.name}
          </CardTitle>
          <Badge variant="outline" className="rounded-md font-normal">
            {asset.version}
          </Badge>
        </CardHeader>
        <CardContent className="px-3 pb-3 pt-2">
          <p className="flex items-center gap-1.5 truncate text-xs text-muted-foreground">
            <AssetTypeIcon kind={kind} className="size-3.5" />
            <span className="truncate">{kindLabel}</span>
            <span aria-hidden="true">·</span>
            <span className="shrink-0">
              {asset.canvasSize.replace(" px", "")}
            </span>
          </p>
        </CardContent>
      </div>
      <div className="absolute top-2 right-2 z-20 flex gap-1">
        <Button
          variant="outline"
          size="icon-sm"
          className="border-border/70 bg-background/90 shadow-xs backdrop-blur-sm"
          aria-label={`Copy ${asset.name}`}
          title="Copy asset"
          onClick={onCopy}
        >
          <Copy />
        </Button>
        <AlertDialog>
          <AlertDialogTrigger
            render={
              <Button
                variant="outline"
                size="icon-sm"
                className="border-border/70 bg-background/90 text-muted-foreground shadow-xs backdrop-blur-sm hover:border-destructive/30 hover:bg-destructive/10 hover:text-destructive"
                aria-label={`Delete ${asset.name}`}
                title="Delete asset"
              />
            }
          >
            <Trash2 />
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {asset.name}?</AlertDialogTitle>
              <AlertDialogDescription>
                This removes the asset from this project. This action cannot be
                undone.
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
