import { useEffect, useRef, useState, type ReactNode } from "react";
import { AlertCircle, Layers3, Ruler, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { AssetTagPicker } from "@/components/asset-tag-picker";
import { DropdownField } from "@/components/ui/custom/dropdown-field";
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
import {
  assetMetadataUpdateSchema,
  assetCanvasSizeOptions,
  type AssetTag,
  type AssetLibraryItem,
  type AssetMetadataUpdate,
} from "@/model/asset";
import {
  isPerspective,
  perspectiveOptions,
  type Perspective,
} from "@/model/project";

import { AssetPreview } from "./asset-preview";

export function AssetEditDialog({
  asset,
  availableTags = [],
  error,
  isSaving,
  onClose,
  onSave,
  projectId,
}: {
  asset?: AssetLibraryItem;
  availableTags?: readonly AssetTag[];
  error?: Error;
  isSaving: boolean;
  onClose: () => void;
  onSave: (metadata: AssetMetadataUpdate) => void;
  projectId?: string;
}) {
  const { t } = useTranslation(["assets", "common"]);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState<AssetTag[]>([]);
  const [canvasSize, setCanvasSize] = useState("");
  const [perspective, setPerspective] = useState<Perspective>(
    perspectiveOptions[0],
  );
  const [validationError, setValidationError] = useState<string>();
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
    setPerspective(asset.perspective);
  }, [asset]);

  const canvasOptions = Array.from(
    new Set([...assetCanvasSizeOptions, canvasSize]),
  );
  const metadataResult = assetMetadataUpdateSchema.safeParse({
    name,
    description,
    tags,
    canvasSize,
    perspective,
  });
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
              if (!metadataResult.success) {
                setValidationError(metadataResult.error.issues[0]?.message);
                return;
              }
              setValidationError(undefined);
              onSave(metadataResult.data);
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
              <span className="sr-only">{t("common:actions.close")}</span>
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
                    {t(`common:assetKinds.${asset.kind}`)}
                    <span aria-hidden="true">/</span>
                    {asset.version}
                  </div>
                  <DialogTitle className="text-xl leading-tight">
                    {t("edit.title")}
                  </DialogTitle>
                  <DialogDescription>{t("edit.description")}</DialogDescription>
                </DialogHeader>

                <div className="mt-6 space-y-5">
                  <Field label={t("edit.name")} htmlFor="asset-name">
                    <Input
                      disabled={isSaving}
                      id="asset-name"
                      value={name}
                      onChange={(event) => {
                        setValidationError(undefined);
                        setName(event.target.value);
                      }}
                    />
                  </Field>
                  <Field
                    label={t("edit.descriptionField")}
                    htmlFor="asset-description"
                  >
                    <Textarea
                      disabled={isSaving}
                      id="asset-description"
                      value={description}
                      onChange={(event) => {
                        setValidationError(undefined);
                        setDescription(event.target.value);
                      }}
                    />
                  </Field>
                  <AssetTagPicker
                    availableTags={availableTags}
                    disabled={isSaving}
                    id="asset-tags"
                    tags={tags}
                    onChange={(nextTags) => {
                      setValidationError(undefined);
                      setTags(nextTags);
                    }}
                  />
                  <div className="grid grid-cols-2 gap-4">
                    <DropdownField
                      disabled={isSaving}
                      label={
                        <>
                          <Ruler className="size-3.5" />
                          {t("edit.canvas")}
                        </>
                      }
                      onChange={(value) => {
                        setValidationError(undefined);
                        setCanvasSize(value);
                      }}
                      options={canvasOptions}
                      size="compact"
                      value={canvasSize}
                    />
                    <DropdownField
                      disabled={isSaving}
                      label={
                        <>
                          <Layers3 className="size-3.5" />
                          {t("edit.perspective")}
                        </>
                      }
                      onChange={(value) => {
                        if (isPerspective(value)) {
                          setValidationError(undefined);
                          setPerspective(value);
                        }
                      }}
                      options={perspectiveOptions}
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
                  {validationError ? (
                    <div
                      className="flex items-start gap-2 border border-destructive/25 bg-destructive/5 px-3 py-2 text-sm text-destructive"
                      role="alert"
                    >
                      <AlertCircle className="mt-0.5 size-4 shrink-0" />
                      <span>{validationError}</span>
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
                {t("common:actions.close")}
              </DialogClose>
              <Button
                type="submit"
                disabled={isSaving || !metadataResult.success}
              >
                {isSaving ? t("edit.saving") : t("common:actions.save")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      ) : null}
    </Dialog>
  );
}

function Field({
  children,
  htmlFor,
  label,
}: {
  children: ReactNode;
  htmlFor: string;
  label: string;
}) {
  return (
    <div className="space-y-2">
      <label
        htmlFor={htmlFor}
        className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground"
      >
        {label}
      </label>
      {children}
    </div>
  );
}
