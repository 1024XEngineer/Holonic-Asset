import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Textarea } from "@/components/ui/textarea";
import type { TilesetAssetCreationDraft } from "../types";
import { ItemShapePicker } from "./item-shape-picker";

const itemCounts = [1, 2, 3, 4, 5, 6, 8];

function createEmptyItem(): TilesetAssetCreationDraft<File>["tiles"][number] {
  return { description: "", reference: undefined, shape: [[0, 0]] };
}

export function TilesetAssetFields({
  draft,
  onChange,
}: {
  draft: TilesetAssetCreationDraft<File>;
  onChange: (draft: TilesetAssetCreationDraft<File>) => void;
}) {
  const [expandedItems, setExpandedItems] = useState(
    () => new Set(draft.tiles.map((_, index) => index)),
  );

  useEffect(
    () => setExpandedItems(new Set(draft.tiles.map((_, index) => index))),
    [draft.tiles.length],
  );

  const updateItems = (tiles: typeof draft.tiles) =>
    onChange({ ...draft, tiles });
  const updateItem = (
    index: number,
    patch: Partial<(typeof draft.tiles)[number]>,
  ) =>
    updateItems(
      draft.tiles.map((item, itemIndex) =>
        itemIndex === index ? { ...item, ...patch } : item,
      ),
    );

  return (
    <>
      <CountSelect
        value={draft.tiles.length}
        onChange={(count) =>
          updateItems(
            Array.from(
              { length: count },
              (_, index) => draft.tiles[index] ?? createEmptyItem(),
            ),
          )
        }
      />
      <div className="grid gap-4">
        {draft.tiles.map((item, index) => {
          const expanded = expandedItems.has(index);

          return (
            <section
              key={index}
              className="grid gap-5 rounded-lg border p-4"
              aria-label={`Tileset item ${index + 1}`}
            >
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-sm font-semibold">Item {index + 1}</h3>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={`${expanded ? "Collapse" : "Expand"} item ${index + 1}`}
                  aria-expanded={expanded}
                  onClick={() =>
                    setExpandedItems((current) => {
                      const next = new Set(current);
                      if (next.has(index)) next.delete(index);
                      else next.add(index);
                      return next;
                    })
                  }
                >
                  <ChevronDown
                    className={`transition-transform ${expanded ? "" : "-rotate-90"}`}
                  />
                </Button>
              </div>
              {expanded ? (
                <>
                  <label className="grid gap-2 text-sm font-medium">
                    Item {index + 1} description
                    <Textarea
                      required
                      className="resize-none"
                      value={item.description}
                      onChange={(event) =>
                        updateItem(index, { description: event.target.value })
                      }
                    />
                  </label>
                  <div className="grid gap-5">
                    <ItemShapePicker
                      shape={item.shape}
                      onChange={(shape) => updateItem(index, { shape })}
                    />
                    <div className="grid gap-2 text-sm font-medium">
                      <span>Reference image</span>
                      <ImageDropzone
                        className="min-h-40"
                        fileName={item.reference?.name}
                        onSelect={(reference) =>
                          updateItem(index, { reference })
                        }
                        onClear={() =>
                          updateItem(index, { reference: undefined })
                        }
                      />
                    </div>
                  </div>
                </>
              ) : null}
            </section>
          );
        })}
      </div>
    </>
  );
}

function CountSelect({
  value,
  onChange,
}: {
  value: number;
  onChange: (count: number) => void;
}) {
  const [open, setOpen] = useState(false);

  return (
    <label className="grid gap-2 text-sm font-medium">
      Item count
      <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="outline"
              className="h-9 w-full justify-between px-3 font-normal"
            />
          }
        >
          {value}
          <ChevronDown className="size-4 text-muted-foreground" />
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-(--anchor-width)">
          <DropdownMenuRadioGroup
            value={String(value)}
            onValueChange={(nextValue) => {
              onChange(Number(nextValue));
              setOpen(false);
            }}
          >
            {itemCounts.map((count) => (
              <DropdownMenuRadioItem key={count} value={String(count)}>
                {count}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </label>
  );
}
