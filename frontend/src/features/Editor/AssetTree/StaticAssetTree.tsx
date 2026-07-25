import {
  BedDouble,
  ChevronDown,
  Fence,
  Grid2X2,
  LampDesk,
  PackageOpen,
} from "lucide-react";
import { useState } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { EditorSpriteSheetItem } from "@/types/record";

const itemIcons = {
  bed: BedDouble,
  lamp: LampDesk,
  fence: Fence,
  object: PackageOpen,
};

export function StaticAssetTree({
  items,
  selectedItems,
  isTileSelected,
  onToggleItem,
  onToggleTile,
}: {
  items: EditorSpriteSheetItem[];
  selectedItems: string[];
  isTileSelected: (itemId: string, tileId: string) => boolean;
  onToggleItem: (item: string) => void;
  onToggleTile: (itemId: string, tileId: string) => void;
}) {
  const [expandedItems, setExpandedItems] = useState<string[]>([]);

  return (
    <aside className="flex w-full shrink-0 flex-col border-b border-black/10 bg-white lg:h-full lg:w-[16.5rem] lg:border-r lg:border-b-0">
      <ScrollArea className="max-h-[15rem] flex-1 lg:max-h-none">
        <div className="p-3">
          <p className="px-2 pb-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#81786d]">
            Exportable objects
          </p>
          <div className="space-y-1">
            {items.map((item) => {
              const Icon = itemIcons[item.icon];
              const expanded = expandedItems.includes(item.id);
              const selected = selectedItems.includes(item.id);

              return (
                <div key={item.id}>
                  <div
                    className={`flex items-center rounded-lg transition-colors ${selected ? "bg-black/5 text-[#2d2923]" : "text-[#51493f] hover:bg-black/[.04]"}`}
                  >
                    <button
                      type="button"
                      aria-pressed={selected}
                      onClick={() => onToggleItem(item.id)}
                      className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left text-xs font-medium"
                    >
                      <Icon className="size-4 text-[#4c7e5e]" />
                      <span className="min-w-0 flex-1 truncate">
                        {item.label}
                      </span>
                      <span className="font-mono text-[10px] text-[#81786d]">
                        {item.tiles.length}
                      </span>
                    </button>
                    <button
                      type="button"
                      aria-label={`${expanded ? "Collapse" : "Expand"} ${item.label} tiles`}
                      aria-expanded={expanded}
                      onClick={() =>
                        setExpandedItems((current) =>
                          current.includes(item.id)
                            ? current.filter(
                                (candidate) => candidate !== item.id,
                              )
                            : [...current, item.id],
                        )
                      }
                      className="mr-1 rounded-md p-1.5 text-[#81786d] hover:bg-black/[.05]"
                    >
                      <ChevronDown
                        className={`size-3.5 transition-transform ${expanded ? "rotate-0" : "-rotate-90"}`}
                      />
                    </button>
                  </div>
                  {expanded ? (
                    <div className="ml-4 grid grid-cols-4 gap-1 border-l border-black/10 py-1 pl-2">
                      {item.tiles.map((tile) => {
                        const tileSelected = isTileSelected(item.id, tile.id);

                        return (
                          <button
                            key={tile.id}
                            type="button"
                            aria-label={`${item.label}: ${tile.label}`}
                            aria-pressed={tileSelected}
                            onClick={() => onToggleTile(item.id, tile.id)}
                            className={`grid aspect-square place-items-center rounded-md border transition-colors ${tileSelected ? "border-[#b86b70] bg-[#b86b70]/10 text-[#8b4e53]" : "border-black/10 text-[#6d8fbd] hover:bg-black/[.04]"}`}
                          >
                            <Grid2X2 className="size-3.5" />
                          </button>
                        );
                      })}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
}
