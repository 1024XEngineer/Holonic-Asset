import { ChevronDown, LoaderCircle, Sparkles } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export type EditorGenerationTask = {
  id: string;
  name: string;
  prompt: string;
  status: "queued" | "processing";
};

export function GenerationTaskDropdown({
  tasks,
}: {
  tasks: EditorGenerationTask[];
}) {
  if (tasks.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="inline-flex h-8 max-w-full items-center gap-2 rounded-lg border border-[#b86b70]/25 bg-[#b86b70]/5 px-2.5 text-xs font-medium text-[#6d4e50] transition-colors hover:bg-[#b86b70]/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#b86b70]">
        <LoaderCircle className="size-3.5 shrink-0 animate-spin text-[#b86b70]" />
        <span className="hidden truncate sm:inline">
          {tasks.length} generation{tasks.length === 1 ? "" : "s"} active
        </span>
        <Badge className="h-4 min-w-4 rounded-full border-[#b86b70]/25 bg-white px-1 text-[10px] text-[#b86b70]">
          {tasks.length}
        </Badge>
        <ChevronDown className="size-3.5 shrink-0" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="center" sideOffset={8} className="w-80 p-0">
        <div className="max-h-72 overflow-y-auto p-1.5">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="flex min-w-0 items-start gap-3 rounded-md px-2 py-2.5"
            >
              <div className="mt-0.5 grid size-7 shrink-0 place-items-center rounded-md bg-[#b86b70]/10 text-[#b86b70]">
                <Sparkles className="size-3.5" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="min-w-0 flex-1 truncate text-xs font-semibold text-[#403a32]">
                    {task.name}
                  </p>
                  <Badge
                    variant="outline"
                    className="shrink-0 border-[#b86b70]/25 text-[10px] text-[#8f6669]"
                  >
                    {task.status === "queued" ? "Queued" : "Generating"}
                  </Badge>
                </div>
                <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-[#81786d]">
                  {task.prompt}
                </p>
              </div>
            </div>
          ))}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
