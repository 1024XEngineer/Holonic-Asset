import {
  ChevronLeft,
  ChevronRight,
  Folder,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { useProjectDetailQuery } from "@/model";
import type { ProjectSummary } from "@/model/project";
import { ProjectSettingsDialog } from "./project-settings-dialog";
import type { ProjectLibraryProjectModel } from "./state/use-project-library";

const PROJECT_SIDEBAR_OPEN_STORAGE_KEY = "game-asset-pack:project-sidebar-open";

export function ProjectSidebar({
  library,
}: {
  library: ProjectLibraryProjectModel;
}) {
  const { t } = useTranslation(["projects", "common"]);
  const [isOpen, setIsOpen] = useState(readProjectSidebarOpen);
  const [editingProjectId, setEditingProjectId] = useState<string>();
  const { create, items, remove, select, selectedId, update } = library;
  const { data: editingProjectDetail } =
    useProjectDetailQuery(editingProjectId);
  const editingProject =
    editingProjectDetail ??
    items.find((project) => project.id === editingProjectId);

  const isSelected = (projectId: string) => projectId === selectedId;
  const projectButton = (project: ProjectSummary, compact = false) => (
    <Button
      key={project.id}
      type="button"
      aria-label={compact ? project.name : undefined}
      aria-current={isSelected(project.id) ? "page" : undefined}
      variant={
        compact ? (isSelected(project.id) ? "default" : "outline") : "ghost"
      }
      size={compact ? "icon-lg" : undefined}
      className={
        compact
          ? undefined
          : "min-w-0 flex-1 justify-start rounded-md px-1.5 py-1"
      }
      onClick={() => void select(project.id)}
    >
      <Folder className="size-4 shrink-0 text-muted-foreground" />
      {compact ? null : (
        <span className="min-w-0 flex-1 text-left">
          <span className="block truncate text-sm font-semibold">
            {project.name}
          </span>
          <span className="block truncate text-xs text-muted-foreground">
            {project.style}
          </span>
        </span>
      )}
    </Button>
  );

  return (
    <aside
      className={cn(
        "w-16 shrink-0 border-r bg-sidebar transition-[width] duration-300 ease-out",
        isOpen && "md:w-80",
      )}
    >
      <div className="flex h-full min-h-0 flex-col">
        <div className="flex h-16 items-center justify-center px-3 md:justify-between">
          {isOpen ? (
            <div className="hidden min-w-0 md:block">
              <p className="text-xs font-semibold uppercase text-muted-foreground">
                {t("project.sidebar.projects")}
              </p>
              <h2 className="truncate text-lg font-semibold">
                {t("assetLibrary")}
              </h2>
            </div>
          ) : null}
          <Button
            aria-label={
              isOpen
                ? t("project.sidebar.collapseList")
                : t("project.sidebar.expandList")
            }
            variant="outline"
            size="icon-sm"
            className="hidden md:inline-flex"
            onClick={() =>
              setIsOpen((current) => {
                const next = !current;
                writeProjectSidebarOpen(next);
                return next;
              })
            }
          >
            {isOpen ? <ChevronLeft /> : <ChevronRight />}
          </Button>
          <Folder className="size-5 text-muted-foreground md:hidden" />
        </div>
        <Separator />

        {isOpen ? (
          <ScrollArea className="hidden min-h-0 flex-1 px-3 py-4 md:block">
            <Button
              type="button"
              className="mb-4 w-full justify-start"
              onClick={() => void create()}
            >
              <Plus data-icon="inline-start" />
              {t("common:actions.newProject")}
            </Button>
            <div className="space-y-2">
              {items.map((project) => (
                <div
                  key={project.id}
                  className={cn(
                    "group flex items-center gap-1 rounded-lg border bg-card p-1.5 transition-colors hover:bg-muted",
                    isSelected(project.id)
                      ? "border-foreground shadow-sm"
                      : "border-border",
                  )}
                >
                  {projectButton(project)}
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="pointer-events-none opacity-0 text-muted-foreground transition-all group-hover:pointer-events-auto group-hover:opacity-100 hover:bg-foreground/10 hover:text-foreground focus-visible:pointer-events-auto focus-visible:opacity-100 focus-visible:text-foreground"
                    aria-label={t("project.sidebar.edit", {
                      name: project.name,
                    })}
                    onClick={() => setEditingProjectId(project.id)}
                  >
                    <Pencil />
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger
                      render={
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="pointer-events-none opacity-0 text-muted-foreground transition-all group-hover:pointer-events-auto group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive focus-visible:pointer-events-auto focus-visible:opacity-100 focus-visible:text-destructive"
                          aria-label={t("project.sidebar.delete", {
                            name: project.name,
                          })}
                        />
                      }
                    >
                      <Trash2 />
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>
                          {t("project.sidebar.deleteConfirm", {
                            name: project.name,
                          })}
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                          {t("project.sidebar.deleteDescription")}
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>
                          {t("common:actions.cancel")}
                        </AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => void remove(project.id)}
                        >
                          {t("project.sidebar.deleteAction")}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          </ScrollArea>
        ) : (
          <div className="hidden min-h-0 flex-1 flex-col items-center gap-3 overflow-y-auto py-4 md:flex">
            <Button
              type="button"
              aria-label={t("common:actions.newProject")}
              variant="outline"
              size="icon-lg"
              onClick={() => void create()}
            >
              <Plus />
            </Button>
            {items.map((project) => projectButton(project, true))}
          </div>
        )}

        <div className="flex min-h-0 flex-1 flex-col items-center gap-3 overflow-y-auto py-4 md:hidden">
          <Button
            type="button"
            aria-label={t("common:actions.newProject")}
            variant="outline"
            size="icon-lg"
            onClick={() => void create()}
          >
            <Plus />
          </Button>
          {items.map((project) => projectButton(project, true))}
        </div>
      </div>
      {editingProject ? (
        <ProjectSettingsDialog
          key={editingProject.id}
          project={editingProject}
          onOpenChange={(open) => {
            if (!open) setEditingProjectId(undefined);
          }}
          onSave={update}
        />
      ) : null}
    </aside>
  );
}

function readProjectSidebarOpen() {
  try {
    return localStorage.getItem(PROJECT_SIDEBAR_OPEN_STORAGE_KEY) !== "false";
  } catch {
    return true;
  }
}

function writeProjectSidebarOpen(isOpen: boolean) {
  try {
    localStorage.setItem(PROJECT_SIDEBAR_OPEN_STORAGE_KEY, String(isOpen));
  } catch {
    // The sidebar remains usable when browser storage is unavailable.
  }
}
