import { PanelsTopLeft, Type, MousePointerClick } from "lucide-react";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { EditorUiComponent } from "../../domain";

const componentIcons = {
  panel: PanelsTopLeft,
  label: Type,
  button: MousePointerClick,
};

export function UiComponentTree({
  components,
  selectedComponentIds,
  onToggleComponent,
}: {
  components: EditorUiComponent[];
  selectedComponentIds: string[];
  onToggleComponent: (componentId: string) => void;
}) {
  return (
    <aside className="flex w-full shrink-0 flex-col border-b border-black/10 bg-white lg:h-full lg:w-[16.5rem] lg:border-r lg:border-b-0">
      <ScrollArea className="max-h-[15rem] flex-1 lg:max-h-none">
        <div className="p-3">
          <p className="px-2 pb-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#81786d]">
            UI components
          </p>
          <div className="space-y-1">
            {components.map((component) => {
              const Icon = componentIcons[component.kind];
              const selected = selectedComponentIds.includes(component.id);

              return (
                <button
                  key={component.id}
                  type="button"
                  aria-pressed={selected}
                  onClick={() => onToggleComponent(component.id)}
                  className={`flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-xs font-medium transition-colors ${selected ? "bg-black/5 text-[#2d2923]" : "text-[#51493f] hover:bg-black/[.04]"}`}
                >
                  <Icon className="size-4 text-[#b86b70]" />
                  <span className="min-w-0 flex-1 truncate">
                    {component.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
}
