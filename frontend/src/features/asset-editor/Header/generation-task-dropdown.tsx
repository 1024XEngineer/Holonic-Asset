import {
  AlertCircle,
  Check,
  ChevronDown,
  LoaderCircle,
  Sparkles,
  X,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export type EditorGenerationTask = {
  id: string;
  name: string;
  prompt: string;
  status: "queued" | "processing" | "awaiting_application" | "failed";
  error?: string;
  isResolving?: boolean;
  onApply?: () => void;
  onDiscard?: () => void;
};

export function GenerationTaskDropdown({
  tasks,
}: {
  tasks: EditorGenerationTask[];
}) {
  const { t } = useTranslation("editor");
  if (tasks.length === 0) return null;
  const hasActiveTasks = tasks.some(
    (task) => task.status === "queued" || task.status === "processing",
  );
  const hasAwaitingTasks = tasks.some(
    (task) => task.status === "awaiting_application",
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="inline-flex h-8 max-w-full items-center gap-2 rounded-lg border border-primary/25 bg-primary/5 px-2.5 text-xs font-medium text-primary transition-colors hover:bg-primary/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
        {hasActiveTasks ? (
          <LoaderCircle className="size-3.5 shrink-0 animate-spin" />
        ) : hasAwaitingTasks ? (
          <Sparkles className="size-3.5 shrink-0" />
        ) : (
          <AlertCircle className="size-3.5 shrink-0 text-destructive" />
        )}
        <span className="hidden truncate sm:inline">
          {t(
            hasActiveTasks
              ? "generationActive"
              : hasAwaitingTasks
                ? "generationReady"
                : "generationFailed",
            {
              count: tasks.length,
            },
          )}
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
          {tasks.map((task) => (
            <div
              key={task.id}
              className="flex min-w-0 items-start gap-3 rounded-md px-2 py-2.5"
            >
              <div className="mt-0.5 grid size-7 shrink-0 place-items-center rounded-md bg-primary/10 text-primary">
                {task.status === "failed" ? (
                  <AlertCircle className="size-3.5 text-destructive" />
                ) : (
                  <Sparkles className="size-3.5" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="min-w-0 flex-1 truncate text-xs font-semibold">
                    {task.name}
                  </p>
                  <Badge
                    variant={
                      task.status === "failed" ? "destructive" : "outline"
                    }
                    className="shrink-0 text-[10px] text-muted-foreground"
                  >
                    {task.status === "queued"
                      ? t("queued")
                      : task.status === "failed"
                        ? t("failed")
                        : task.status === "awaiting_application"
                          ? t("awaitingApplication")
                          : t("generating")}
                  </Badge>
                </div>
                <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-muted-foreground">
                  {task.prompt}
                </p>
                {task.error ? (
                  <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-destructive">
                    {task.error}
                  </p>
                ) : null}
                {task.status === "awaiting_application" ? (
                  <div className="mt-2 flex items-center gap-1.5">
                    <Button
                      size="xs"
                      disabled={task.isResolving}
                      onClick={task.onApply}
                    >
                      <Check data-icon="inline-start" />
                      {t("applyGeneration")}
                    </Button>
                    <Button
                      size="xs"
                      variant="outline"
                      disabled={task.isResolving}
                      onClick={task.onDiscard}
                    >
                      <X data-icon="inline-start" />
                      {t("discardGeneration")}
                    </Button>
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
