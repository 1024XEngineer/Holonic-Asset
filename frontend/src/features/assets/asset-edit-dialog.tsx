import { useEffect, useRef, useState, type ReactNode } from "react";
import { AlertCircle, Layers3, Ruler, Tags, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownField } from "@/components/ui/custom/dropdown-field";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { AssetKindIcon, getAssetKindConfig } from "@/components/asset-kind";
import {
  assetCanvasSizeOptions,
  assetPerspectiveOptions,
  isAssetPerspective,
  type AssetLibraryItem,
  type AssetMetadataUpdate,
} from "@/model/asset";

import { AssetPreview } from "./asset-preview";

export function AssetEditDialog({
  asset,
  error,
  isSaving,
  onClose,
  onSave,
  projectId,
}: {
  asset?: AssetLibraryItem;
  error?: Error;
  isSaving: boolean;
  onClose: () => void;
  onSave: (metadata: AssetMetadataUpdate) => void;
  projectId?: string;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [canvasSize, setCanvasSize] = useState("");
  const [perspective, setPerspective] = useState("");
  const initializedAssetIdRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    if (!asset) {
      initializedAssetIdRef.current = undefined;
      return;
    }
    if (initializedAssetIdRef.current === asset.id) return;

    initializedAssetIdRef.current = asset.id;

    setName(asset.name);
    setDescription(asset.description);
    setTags(asset.tags);
    setCanvasSize(asset.canvasSize);
    setPerspective(
      isAssetPerspective(asset.perspective)
        ? asset.perspective
        : assetPerspectiveOptions[0],
    );
  }, [asset]);

  const tagOptions = Array.from(new Set([...availableTags, ...tags]));
  const canvasOptions = Array.from(
    new Set([...assetCanvasSizeOptions, canvasSize]),
  );
  const toggleTag = (tag: string, checked: boolean) => {
    setTags((currentTags) =>
      checked
        ? [...currentTags, tag]
        : currentTags.filter((currentTag) => currentTag !== tag),
    );
  };

  return (
    <Dialog
      open={Boolean(asset)}
      onOpenChange={(open) => !open && !isSaving && onClose()}
    >
      {asset ? (
        <DialogContent
          className="max-h-[calc(100dvh-2rem)] overflow-y-auto p-0 sm:max-w-3xl"
          showCloseButton={false}
        >
          <form
            className="contents"
            onSubmit={(event) => {
              event.preventDefault();
              onSave({ name, description, tags, canvasSize, perspective });
            }}
          >
            <DialogClose
              render={
                <Button
                  disabled={isSaving}
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="absolute right-2 top-2 z-10 bg-background/80 backdrop-blur-sm"
                />
              }
            >
              <X />
              <span className="sr-only">Close</span>
            </DialogClose>

            <div className="grid sm:grid-cols-[minmax(0,1fr)_minmax(20rem,1fr)]">
              <AssetPreview
                asset={asset}
                className="aspect-square border-b sm:aspect-auto sm:min-h-[34rem] sm:border-b-0 sm:border-r"
                projectId={projectId}
              />
              <div className="min-w-0 p-5 sm:p-6">
                <DialogHeader className="pr-7">
                  <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                    <AssetKindIcon kind={asset.kind} className="size-3.5" />
                    {getAssetKindConfig(asset.kind).label}
                    <span aria-hidden="true">/</span>
                    {asset.version}
                  </div>
                  <DialogTitle className="text-xl leading-tight">
                    Edit asset
                  </DialogTitle>
                  <DialogDescription>
                    Update the asset information used throughout this project.
                  </DialogDescription>
                </DialogHeader>

                <div className="mt-6 space-y-5">
                  <Field label="Name" htmlFor="asset-name">
                    <Input
                      disabled={isSaving}
                      id="asset-name"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                    />
                  </Field>
                  <Field label="Description" htmlFor="asset-description">
                    <Textarea
                      disabled={isSaving}
                      id="asset-description"
                      value={description}
                      onChange={(event) => setDescription(event.target.value)}
                    />
                  </Field>
                  <Field
                    label="Tags"
                    htmlFor="asset-tags"
                    icon={<Tags className="size-3.5" />}
                  >
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        render={
                          <Button
                            disabled={isSaving}
                            id="asset-tags"
                            type="button"
                            variant="outline"
                            className="h-8 w-full justify-start truncate font-normal"
                          />
                        }
                      >
                        {tags.length > 0 ? tags.join(", ") : "Select tags"}
                      </DropdownMenuTrigger>
                      <DropdownMenuContent className="w-[var(--anchor-width)] min-w-52">
                        {tagOptions.map((tag) => (
                          <DropdownMenuCheckboxItem
                            key={tag}
                            checked={tags.includes(tag)}
                            closeOnClick={false}
                            onCheckedChange={(checked) =>
                              toggleTag(tag, checked)
                            }
                          >
                            {tag}
                          </DropdownMenuCheckboxItem>
                        ))}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </Field>
                  <div className="grid grid-cols-2 gap-4">
                    <DropdownField
                      disabled={isSaving}
                      label={
                        <>
                          <Ruler className="size-3.5" />
                          Canvas
                        </>
                      }
                      onChange={setCanvasSize}
                      options={canvasOptions}
                      size="compact"
                      value={canvasSize}
                    />
                    <DropdownField
                      disabled={isSaving}
                      label={
                        <>
                          <Layers3 className="size-3.5" />
                          Perspective
                        </>
                      }
                      onChange={(value) => {
                        if (isAssetPerspective(value)) setPerspective(value);
                      }}
                      options={assetPerspectiveOptions}
                      size="compact"
                      value={perspective}
                    />
                  </div>
                  {error ? (
                    <div
                      className="flex items-start gap-2 border border-destructive/25 bg-destructive/5 px-3 py-2 text-sm text-destructive"
                      role="alert"
                    >
                      <AlertCircle className="mt-0.5 size-4 shrink-0" />
                      <span>{error.message}</span>
                    </div>
                  ) : null}
                </div>
              </div>
            </div>
            <DialogFooter className="mx-0 mb-0 rounded-none sm:col-span-2">
              <DialogClose
                render={
                  <Button type="button" variant="outline" disabled={isSaving} />
                }
              >
                Close
              </DialogClose>
              <Button type="submit" disabled={isSaving || !name.trim()}>
                {isSaving ? "Saving..." : "Save changes"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

const availableTags = [
  "pixel-art",
  "character",
  "object",
  "environment",
  "interface",
  "terrain",
  "top-down",
];

function Field({
  children,
  htmlFor,
  icon,
  label,
}: {
  children: ReactNode;
  htmlFor: string;
  icon?: ReactNode;
  label: string;
}) {
  return (
    <div className="space-y-2">
      <label
        htmlFor={htmlFor}
        className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground"
      >
        {icon}
        {label}
      </label>
      {children}
    </div>
  );
}
