import { ChevronDown, PackageOpen } from "lucide-react";
import { useState } from "react";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { EditorSpriteSheetItem } from "../../domain";

export function StaticAssetTree({
  items,
  selectedItems,
  isCellSelected,
  onToggleItem,
  onToggleCell,
}: {
  items: EditorSpriteSheetItem[];
  selectedItems: string[];
  isCellSelected: (itemId: string, cellIndex: number) => boolean;
  onToggleItem: (itemId: string) => void;
  onToggleCell: (itemId: string, cellIndex: number) => void;
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
              const expanded = expandedItems.includes(item.id);
              const selected = selectedItems.includes(item.id);
              const layout = getCellLayout(item);

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
                      <PackageOpen className="size-4 text-[#4c7e5e]" />
                      <span className="min-w-0 flex-1 truncate">{item.label}</span>
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
                            ? current.filter((candidate) => candidate !== item.id)
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
                    <div
                      aria-label={`${item.label} tile layout`}
                      className="ml-8 mt-2 grid w-fit overflow-visible bg-white outline outline-1 outline-[#5dabb0]/80"
                      style={{
                        width: `${layout.width * 44}px`,
                        height: `${layout.height * 44}px`,
                        gridTemplateColumns: `repeat(${layout.width}, 44px)`,
                        gridTemplateRows: `repeat(${layout.height}, 44px)`,
                        backgroundImage:
                          "repeating-linear-gradient(to right, rgb(93 171 176 / 80%) 0 1px, transparent 1px 44px), repeating-linear-gradient(to bottom, rgb(93 171 176 / 80%) 0 1px, transparent 1px 44px)",
                      }}
                    >
                      {item.tiles.map(([x, y], cellIndex) => {
                        const tileSelected = isCellSelected(item.id, cellIndex);

                        return (
                          <button
                            key={`${x}:${y}`}
                            type="button"
                            aria-label={`${item.label}: Tile ${cellIndex + 1}`}
                            aria-pressed={tileSelected}
                            onClick={() => onToggleCell(item.id, cellIndex)}
                            style={{
                              gridColumn: x - layout.minX + 1,
                              gridRow: y - layout.minY + 1,
                            }}
                            className={`z-10 size-11 border-0 transition-colors ${tileSelected ? "bg-[#b86b70]/25" : "hover:bg-black/5"}`}
                          />
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

function getCellLayout(item: EditorSpriteSheetItem) {
  const minX = Math.min(...item.tiles.map(([x]) => x));
  const minY = Math.min(...item.tiles.map(([, y]) => y));
  const maxX = Math.max(...item.tiles.map(([x]) => x));
  const maxY = Math.max(...item.tiles.map(([, y]) => y));

  return { minX, minY, width: maxX - minX + 1, height: maxY - minY + 1 };
}
