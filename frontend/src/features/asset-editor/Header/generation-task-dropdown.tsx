import { AlertCircle, ChevronDown, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  GenerationTaskList,
  type GenerationTaskListItem,
} from "@/features/generation";

export function GenerationTaskDropdown({
  tasks,
}: {
  tasks: GenerationTaskListItem[];
}) {
  const { t } = useTranslation("editor");
  if (tasks.length === 0) return null;
  const hasActiveTasks = tasks.some((task) => task.status !== "failed");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="inline-flex h-8 max-w-full items-center gap-2 rounded-lg border border-primary/25 bg-primary/5 px-2.5 text-xs font-medium text-primary transition-colors hover:bg-primary/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
        {hasActiveTasks ? (
          <LoaderCircle className="size-3.5 shrink-0 animate-spin" />
        ) : (
          <AlertCircle className="size-3.5 shrink-0 text-destructive" />
        )}
        <span className="hidden truncate sm:inline">
          {t(hasActiveTasks ? "generationActive" : "generationFailed", {
            count: tasks.length,
          })}
        </span>
        <Badge
          variant="outline"
          className="h-4 min-w-4 rounded-full border-primary/25 bg-background px-1 text-[10px] text-primary"
        >
          {tasks.length}
        </Badge>
        <ChevronDown className="size-3.5 shrink-0" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="center" sideOffset={8} className="w-80 p-0">
        <div className="max-h-72 overflow-y-auto p-1.5">
          <GenerationTaskList tasks={tasks} variant="dropdown" />
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
