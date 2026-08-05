import { useEffect, useState, type ReactNode } from "react";
import {
  AlertCircle,
  ChevronDown,
  Layers3,
  Ruler,
  Tags,
  X,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
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
import { AssetKindIcon } from "@/components/asset-kind";
import type { AssetMetadataUpdate } from "@/model/asset";

import { AssetPreview } from "./asset-preview";
import type { AssetLibraryItem } from "./types/asset";

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

  useEffect(() => {
    if (!asset) return;

    setName(asset.name);
    setDescription(asset.description);
    setTags(asset.tags);
    setCanvasSize(asset.canvasSize);
    setPerspective(asset.perspective);
  }, [asset]);

  const tagOptions = Array.from(new Set([...availableTags, ...tags]));
  const canvasOptions = Array.from(new Set([...canvasSizeOptions, canvasSize]));
  const perspectives = Array.from(
    new Set([...perspectiveOptions, perspective]),
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
                accentClassName={asset.accentClassName}
                assetId={asset.id}
                className="aspect-square border-b sm:aspect-auto sm:min-h-[34rem] sm:border-b-0 sm:border-r"
                kind={asset.kind}
                name={asset.name}
                previewCrop={asset.previewCrop}
                previewFrame={asset.previewFrame}
                previewOffset={asset.previewOffset}
                previewScale={asset.previewScale}
                projectId={projectId}
                thumbnailUrl={asset.thumbnailUrl}
              />
              <div className="min-w-0 p-5 sm:p-6">
                <DialogHeader className="pr-7">
                  <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                    <AssetKindIcon kind={asset.kind} className="size-3.5" />
                    {asset.kindLabel}
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
                            className="h-8 w-full justify-between font-normal"
                          />
                        }
                      >
                        {tags.length > 0
                          ? `${tags.length} tag${tags.length === 1 ? "" : "s"} selected`
                          : "Select tags"}
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
                    {tags.length > 0 ? (
                      <div className="flex flex-wrap gap-1.5 pt-1">
                        {tags.map((tag) => (
                          <Badge key={tag} variant="secondary">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    ) : null}
                  </Field>
                  <div className="grid grid-cols-2 gap-4">
                    <Field
                      label="Canvas"
                      htmlFor="asset-canvas"
                      icon={<Ruler className="size-3.5" />}
                    >
                      <SingleSelect
                        disabled={isSaving}
                        id="asset-canvas"
                        value={canvasSize}
                        options={canvasOptions}
                        onValueChange={setCanvasSize}
                      />
                    </Field>
                    <Field
                      label="Perspective"
                      htmlFor="asset-perspective"
                      icon={<Layers3 className="size-3.5" />}
                    >
                      <SingleSelect
                        disabled={isSaving}
                        id="asset-perspective"
                        value={perspective}
                        options={perspectives}
                        onValueChange={setPerspective}
                      />
                    </Field>
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

const canvasSizeOptions = [
  "16 × 16 px",
  "32 × 32 px",
  "48 × 64 px",
  "64 × 64 px",
  "128 × 128 px",
  "320 × 180 px",
  "1920 × 1080 px",
];

const perspectiveOptions = [
  "Top-down",
  "Side view",
  "Screen space",
  "Isometric",
];

function SingleSelect({
  disabled,
  id,
  onValueChange,
  options,
  value,
}: {
  disabled: boolean;
  id: string;
  onValueChange: (value: string) => void;
  options: string[];
  value: string;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            disabled={disabled}
            id={id}
            type="button"
            variant="outline"
            className="h-8 w-full justify-between px-2.5 font-normal"
          />
        }
      >
        <span className="truncate">{value || "Not specified"}</span>
        <ChevronDown className="text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-[var(--anchor-width)] min-w-40">
        <DropdownMenuRadioGroup value={value} onValueChange={onValueChange}>
          {options.map((option) => (
            <DropdownMenuRadioItem key={option} value={option}>
              {option}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

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
