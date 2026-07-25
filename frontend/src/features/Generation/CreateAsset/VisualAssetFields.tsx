import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { Input } from "@/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/custom/native-select";
import type { VisualAssetCreationDraft } from "@/modules/asset-creation";
import type { CreationRequest } from "@/types/generation";

export function VisualAssetFields({
  draft,
  onChange,
}: {
  draft: VisualAssetCreationDraft;
  onChange: (draft: VisualAssetCreationDraft) => void;
}) {
  const isSpriteAsset = draft.kind === "character" || draft.kind === "object";

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="grid gap-2 text-sm font-medium">
          Canvas size
          <Input
            value={draft.canvasSize}
            onChange={(event) =>
              onChange({ ...draft, canvasSize: event.target.value })
            }
          />
        </label>
        <div className="grid gap-2 text-sm font-medium">
          <label htmlFor="create-asset-perspective">Perspective</label>
          <NativeSelect
            id="create-asset-perspective"
            className="w-full"
            value={draft.perspective}
            onChange={(event) =>
              onChange({
                ...draft,
                perspective: event.target.value as NonNullable<
                  CreationRequest["perspective"]
                >,
              })
            }
          >
            <NativeSelectOption value="top-down">Top down</NativeSelectOption>
            <NativeSelectOption value="side-on">Side on</NativeSelectOption>
            <NativeSelectOption value="isometric">Isometric</NativeSelectOption>
          </NativeSelect>
        </div>
      </div>

      {isSpriteAsset ? (
        <>
          <div className="grid gap-2 text-sm font-medium">
            <label htmlFor="create-asset-direction-count">
              Direction count
            </label>
            <NativeSelect
              id="create-asset-direction-count"
              className="w-full"
              value={draft.directionCount}
              onChange={(event) =>
                onChange({
                  ...draft,
                  directionCount: event.target.value as NonNullable<
                    CreationRequest["directionCount"]
                  >,
                })
              }
            >
              <NativeSelectOption value="1">1 direction</NativeSelectOption>
              <NativeSelectOption value="4">4 directions</NativeSelectOption>
              <NativeSelectOption value="8">8 directions</NativeSelectOption>
            </NativeSelect>
          </div>

          <div className="grid gap-2 text-sm font-medium">
            <span>Reference</span>
            <ImageDropzone
              fileName={draft.reference?.name}
              onSelect={(reference) => onChange({ ...draft, reference })}
              onClear={() => onChange({ ...draft, reference: undefined })}
            />
          </div>
        </>
      ) : null}
    </>
  );
}
