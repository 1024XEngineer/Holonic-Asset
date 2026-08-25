import { AlertCircle, LoaderCircle, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AssetKindIcon } from "@/components/asset-kind";
import { Badge } from "@/components/ui/badge";
import type { AssetKind, GenerationRun } from "@/model";

import { GenerationRunRecoveryActions } from "./generation-run-recovery-actions";

export type GenerationTaskListItem = Pick<
  GenerationRun,
  "id" | "name" | "prompt" | "status"
> & {
  projectId?: string;
  kind?: AssetKind;
  error?: string;
};

export function GenerationTaskList({
  tasks,
  variant,
}: {
  tasks: GenerationTaskListItem[];
  variant: "queue" | "dropdown";
}) {
  return tasks.map((task) => (
    <GenerationTaskListRow key={task.id} task={task} variant={variant} />
  ));
}

function GenerationTaskListRow({
  task,
  variant,
}: {
  task: GenerationTaskListItem;
  variant: "queue" | "dropdown";
}) {
  const isFailed = task.status === "failed";
  const isDropdown = variant === "dropdown";
  const recoveryTarget =
    isFailed && task.projectId
      ? { projectId: task.projectId, runId: task.id }
      : undefined;

  if (isDropdown) {
    return (
      <div className="flex min-w-0 items-start gap-3 rounded-md px-2 py-2.5">
        <TaskIcon task={task} variant={variant} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="min-w-0 flex-1 truncate text-xs font-semibold">
              {task.name}
            </p>
            <TaskStatusBadge task={task} variant={variant} />
          </div>
          <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-muted-foreground">
            {task.prompt}
          </p>
          {task.error ? (
            <p className="mt-1 line-clamp-2 text-[11px] leading-4 text-destructive">
              {task.error}
            </p>
          ) : null}
          {recoveryTarget ? (
            <GenerationRunRecoveryActions
              name={task.name}
              target={recoveryTarget}
              variant="dropdown"
            />
          ) : null}
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-3 py-3 sm:flex-row sm:items-center">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <TaskIcon task={task} variant={variant} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{task.name}</p>
          <p className="truncate text-xs text-muted-foreground">
            {task.prompt}
          </p>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2 self-end sm:self-auto">
        <TaskStatusBadge task={task} variant={variant} />
        {recoveryTarget ? (
          <GenerationRunRecoveryActions
            name={task.name}
            target={recoveryTarget}
          />
        ) : null}
      </div>
    </div>
  );
}

function TaskIcon({
  task,
  variant,
}: {
  task: GenerationTaskListItem;
  variant: "queue" | "dropdown";
}) {
  const isDropdown = variant === "dropdown";
  return (
    <span
      className={
        isDropdown
          ? "mt-0.5 grid size-7 shrink-0 place-items-center rounded-md bg-primary/10 text-primary"
          : "grid size-8 shrink-0 place-items-center rounded-md bg-muted text-muted-foreground"
      }
    >
      {task.kind ? (
        <AssetKindIcon
          kind={task.kind}
          className={isDropdown ? "size-3.5" : "size-4"}
        />
      ) : task.status === "failed" ? (
        <AlertCircle className="size-3.5 text-destructive" />
      ) : (
        <Sparkles className="size-3.5" />
      )}
    </span>
  );
}

function TaskStatusBadge({
  task,
  variant,
}: {
  task: GenerationTaskListItem;
  variant: "queue" | "dropdown";
}) {
  const { t } = useTranslation("generation");
  const isFailed = task.status === "failed";
  return (
    <Badge
      variant={isFailed ? "destructive" : "outline"}
      className={
        variant === "dropdown"
          ? "shrink-0 text-[10px] text-muted-foreground"
          : undefined
      }
    >
      {variant === "queue" ? (
        isFailed ? (
          <AlertCircle />
        ) : (
          <LoaderCircle className="animate-spin" />
        )
      ) : null}
      {t(`status.${task.status}`)}
    </Badge>
  );
}
