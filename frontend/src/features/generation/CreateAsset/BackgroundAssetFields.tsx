import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { Input } from "@/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/custom/native-select";
import { Textarea } from "@/components/ui/textarea";
import type {
  SceneryAssetCreationDraft,
  TilesetAssetCreationDraft,
} from "@/model";

const itemCounts = [1, 2, 3, 4, 5, 6, 8];

export function SceneryAssetFields({
  draft,
  onChange,
}: {
  draft: SceneryAssetCreationDraft<File>;
  onChange: (draft: SceneryAssetCreationDraft<File>) => void;
}) {
  return <SceneryFields draft={draft} onChange={onChange} />;
}

export function TilesetAssetFields({
  draft,
  onChange,
}: {
  draft: TilesetAssetCreationDraft<File>;
  onChange: (draft: TilesetAssetCreationDraft<File>) => void;
}) {
  return <TilesetFields draft={draft} onChange={onChange} />;
}

function SceneryFields({
  draft,
  onChange,
}: {
  draft: SceneryAssetCreationDraft<File>;
  onChange: (draft: SceneryAssetCreationDraft<File>) => void;
}) {
  return (
    <>
      <label className="grid gap-2 text-sm font-medium">
        Style
        <Textarea
          required
          className="min-h-20 resize-y"
          placeholder="Describe the overall scene style..."
          value={draft.style}
          onChange={(event) =>
            onChange({ ...draft, style: event.target.value })
          }
        />
      </label>
      <CountSelect
        label="Layer num"
        value={draft.layers.length}
        onChange={(count) =>
          onChange({
            ...draft,
            layers: Array.from(
              { length: count },
              (_, index) => draft.layers[index] ?? { description: "" },
            ),
          })
        }
      />
      <div className="grid gap-3">
        {draft.layers.map((layer, index) => (
          <label key={index} className="grid gap-2 text-sm font-medium">
            Layer {index + 1} description
            <Textarea
              required
              value={layer.description}
              onChange={(event) =>
                onChange({
                  ...draft,
                  layers: draft.layers.map((item, itemIndex) =>
                    itemIndex === index
                      ? { ...item, description: event.target.value }
                      : item,
                  ),
                })
              }
            />
          </label>
        ))}
      </div>
      <label className="grid gap-2 text-sm font-medium">
        Aspect ratio
        <Input
          required
          value={draft.aspectRatio}
          onChange={(event) =>
            onChange({ ...draft, aspectRatio: event.target.value })
          }
          placeholder="e.g. 16:9"
        />
      </label>
      <ImageDropzone
        fileName={draft.reference?.name}
        onSelect={(reference) => onChange({ ...draft, reference })}
        onClear={() => onChange({ ...draft, reference: undefined })}
      />
    </>
  );
}

function TilesetFields({
  draft,
  onChange,
}: {
  draft: TilesetAssetCreationDraft<File>;
  onChange: (draft: TilesetAssetCreationDraft<File>) => void;
}) {
  const updateTile = (
    index: number,
    patch: Partial<TilesetAssetCreationDraft<File>["tiles"][number]>,
  ) =>
    onChange({
      ...draft,
      tiles: draft.tiles.map((tile, tileIndex) =>
        tileIndex === index ? { ...tile, ...patch } : tile,
      ),
    });

  return (
    <>
      <CountSelect
        label="Tile num"
        value={draft.tiles.length}
        onChange={(count) =>
          onChange({
            ...draft,
            tiles: Array.from(
              { length: count },
              (_, index) =>
                draft.tiles[index] ?? {
                  description: "",
                  reference: undefined,
                },
            ),
          })
        }
      />
      <div className="grid gap-4">
        {draft.tiles.map((tile, index) => (
          <div key={index} className="grid gap-2 rounded-lg border p-3">
            <label className="grid gap-2 text-sm font-medium">
              Tile {index + 1} description
              <Textarea
                required
                value={tile.description}
                onChange={(event) =>
                  updateTile(index, { description: event.target.value })
                }
              />
            </label>
            <ImageDropzone
              fileName={tile.reference?.name}
              onSelect={(reference) => updateTile(index, { reference })}
              onClear={() => updateTile(index, { reference: undefined })}
            />
          </div>
        ))}
      </div>
    </>
  );
}

function CountSelect({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (count: number) => void;
}) {
  return (
    <label className="grid gap-2 text-sm font-medium">
      {label}
      <NativeSelect
        className="w-full"
        value={String(value)}
        onChange={(event) => onChange(Number(event.target.value))}
      >
        {itemCounts.map((count) => (
          <NativeSelectOption key={count} value={String(count)}>
            {count}
          </NativeSelectOption>
        ))}
      </NativeSelect>
    </label>
  );
}
