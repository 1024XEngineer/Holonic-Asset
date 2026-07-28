import { ChevronLeft, ChevronRight, Folder, Plus, Trash2 } from "lucide-react";
import { useState } from "react";

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
import type { ProjectSummary } from "@/model";
import { ProjectSettingsDialog } from "./ProjectSettingsDialog";

type ProjectSidebarProps = {
  isProjectRoute: boolean;
  onCreateProject: () => void;
  onDeleteProject: (projectId: string) => void;
  onSelectProject: (projectId: string) => void;
  onUpdateProject: (project: ProjectSummary) => void;
  projects: ProjectSummary[];
  selectedProjectId?: string;
};

export function ProjectSidebar({
  isProjectRoute,
  onCreateProject,
  onDeleteProject,
  onSelectProject,
  onUpdateProject,
  projects,
  selectedProjectId,
}: ProjectSidebarProps) {
  const [isOpen, setIsOpen] = useState(true);

  const isSelected = (projectId: string) =>
    isProjectRoute && projectId === selectedProjectId;
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
      onClick={() => onSelectProject(project.id)}
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
                Projects
              </p>
              <h2 className="truncate text-lg font-semibold">Asset Library</h2>
            </div>
          ) : null}
          <Button
            aria-label={
              isOpen ? "Collapse project list" : "Expand project list"
            }
            variant="outline"
            size="icon-sm"
            className="hidden md:inline-flex"
            onClick={() => setIsOpen((current) => !current)}
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
              onClick={onCreateProject}
            >
              <Plus data-icon="inline-start" />
              New Project
            </Button>
            <div className="space-y-2">
              {projects.map((project) => (
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
                  <ProjectSettingsDialog
                    project={project}
                    onSave={onUpdateProject}
                    iconOnly
                  />
                  <AlertDialog>
                    <AlertDialogTrigger
                      render={
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="pointer-events-none opacity-0 text-muted-foreground transition-all group-hover:pointer-events-auto group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive focus-visible:pointer-events-auto focus-visible:opacity-100 focus-visible:text-destructive"
                          aria-label={`Delete ${project.name}`}
                        />
                      }
                    >
                      <Trash2 />
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>
                          Delete {project.name}?
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                          This removes the project and its asset workspace from
                          this browser. This action cannot be undone.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => onDeleteProject(project.id)}
                        >
                          Delete project
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          </ScrollArea>
        ) : (
          <div className="hidden flex-1 flex-col items-center gap-3 py-4 md:flex">
            <Button
              type="button"
              aria-label="New Project"
              variant="outline"
              size="icon-lg"
              onClick={onCreateProject}
            >
              <Plus />
            </Button>
            {projects.map((project) => projectButton(project, true))}
          </div>
        )}

        <div className="flex flex-1 flex-col items-center gap-3 py-4 md:hidden">
          <Button
            type="button"
            aria-label="New Project"
            variant="outline"
            size="icon-lg"
            onClick={onCreateProject}
          >
            <Plus />
          </Button>
          {projects.map((project) => projectButton(project, true))}
        </div>
      </div>
    </aside>
  );
}
