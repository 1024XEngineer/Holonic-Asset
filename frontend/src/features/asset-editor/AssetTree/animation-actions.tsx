import { PencilLine, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { EditorCharacterAnimation } from "@/model";

type ContextMenuState = {
  animation: EditorCharacterAnimation;
  x: number;
  y: number;
};

type AnimationActionsProps = {
  onRename: (animationId: string, label: string) => void;
  onDelete: (animationId: string) => void;
};

export function useAnimationActions({
  onRename,
  onDelete,
}: AnimationActionsProps) {
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [editingAnimation, setEditingAnimation] =
    useState<EditorCharacterAnimation | null>(null);
  const [editedAnimationLabel, setEditedAnimationLabel] = useState("");

  useEffect(() => {
    if (!contextMenu) return;

    const closeContextMenu = () => setContextMenu(null);
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") closeContextMenu();
    };

    window.addEventListener("pointerdown", closeContextMenu);
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      window.removeEventListener("pointerdown", closeContextMenu);
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [contextMenu]);

  const openContextMenu = (
    event: React.MouseEvent<HTMLElement>,
    animation: EditorCharacterAnimation,
  ) => {
    event.preventDefault();
    setContextMenu({
      animation,
      x: Math.max(8, Math.min(event.clientX, window.innerWidth - 208)),
      y: Math.max(8, Math.min(event.clientY, window.innerHeight - 104)),
    });
  };

  const openEditDialog = (animation: EditorCharacterAnimation) => {
    setContextMenu(null);
    setEditingAnimation(animation);
    setEditedAnimationLabel(animation.label);
  };

  const handleEdit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const label = editedAnimationLabel.trim();
    if (!editingAnimation || !label) return;

    onRename(editingAnimation.id, label);
    setEditingAnimation(null);
  };

  const handleDelete = (animation: EditorCharacterAnimation) => {
    setContextMenu(null);
    onDelete(animation.id);
  };

  return {
    openContextMenu,
    actions: (
      <>
        <Dialog
          open={Boolean(editingAnimation)}
          onOpenChange={(open) => {
            if (!open) setEditingAnimation(null);
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Edit animation information</DialogTitle>
              <DialogDescription>
                Update the name used for this animation in the asset tree.
              </DialogDescription>
            </DialogHeader>
            <form className="grid gap-5" onSubmit={handleEdit}>
              <label
                className="grid gap-2 text-sm font-medium"
                htmlFor="edit-animation-name"
              >
                Animation name
                <Input
                  id="edit-animation-name"
                  autoFocus
                  required
                  value={editedAnimationLabel}
                  onChange={(event) =>
                    setEditedAnimationLabel(event.target.value)
                  }
                />
              </label>
              <DialogFooter>
                <DialogClose
                  render={<Button type="button" variant="outline" />}
                >
                  Cancel
                </DialogClose>
                <Button type="submit" disabled={!editedAnimationLabel.trim()}>
                  Save information
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
        {contextMenu ? (
          <div
            role="menu"
            aria-label={`${contextMenu.animation.label} actions`}
            style={{ left: contextMenu.x, top: contextMenu.y }}
            onPointerDown={(event) => event.stopPropagation()}
            className="fixed z-50 grid w-48 gap-1 rounded-lg border border-black/10 bg-white p-1.5 shadow-lg"
          >
            <button
              type="button"
              role="menuitem"
              onClick={() => openEditDialog(contextMenu.animation)}
              className="flex h-8 items-center gap-2 rounded-md px-2 text-left text-xs font-medium text-[#51493f] hover:bg-black/[.05]"
            >
              <PencilLine className="size-3.5" />
              Edit information
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => handleDelete(contextMenu.animation)}
              className="flex h-8 items-center gap-2 rounded-md px-2 text-left text-xs font-medium text-[#b8565a] hover:bg-[#b8565a]/10"
            >
              <Trash2 className="size-3.5" />
              Delete
            </button>
          </div>
        ) : null}
      </>
    ),
  };
}
