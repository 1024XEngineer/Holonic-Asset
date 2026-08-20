import { ChevronDown } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { TilesetAssetCreationDraft } from "../types";
import { ItemShapePicker } from "./item-shape-picker";

const itemCounts = [1, 2, 3, 4, 5, 6, 8];

function createEmptyItem(): TilesetAssetCreationDraft["tiles"][number] {
  return { name: "", description: "", shape: [[0, 0]] };
}

export function TilesetAssetFields({
  draft,
  onChange,
}: {
  draft: TilesetAssetCreationDraft;
  onChange: (draft: TilesetAssetCreationDraft) => void;
}) {
  const { t } = useTranslation("generation");
  const [expandedItems, setExpandedItems] = useState(
    () => new Set(draft.tiles.map((_, index) => index)),
  );
  const previousTileCountRef = useRef(draft.tiles.length);

  useEffect(() => {
    const previousTileCount = previousTileCountRef.current;
    const nextTileCount = draft.tiles.length;
    if (previousTileCount === nextTileCount) return;

    setExpandedItems((current) => {
      const next = new Set(
        [...current].filter((index) => index < nextTileCount),
      );
      for (let index = previousTileCount; index < nextTileCount; index += 1) {
        next.add(index);
      }
      return next;
    });
    previousTileCountRef.current = nextTileCount;
  }, [draft.tiles.length]);

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
              // Tiles are positional drafts and can only be added or removed at the end.
              // oxlint-disable-next-line react/no-array-index-key
              key={index}
              className="grid gap-5 rounded-lg border p-4"
              aria-label={t("tilesetItem", { number: index + 1 })}
            >
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-sm font-semibold">
                  {t("item", { number: index + 1 })}
                </h3>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t(expanded ? "collapseItem" : "expandItem", {
                    number: index + 1,
                  })}
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
                    {t("itemName", { number: index + 1 })}
                    <Input
                      required
                      placeholder={t("itemNamePlaceholder")}
                      value={item.name}
                      onChange={(event) =>
                        updateItem(index, { name: event.target.value })
                      }
                    />
                  </label>
                  <label className="grid gap-2 text-sm font-medium">
                    {t("itemDescription", { number: index + 1 })}
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
  const { t } = useTranslation("generation");
  const [open, setOpen] = useState(false);

  return (
    <label className="grid gap-2 text-sm font-medium">
      {t("itemCount")}
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
