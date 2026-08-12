import { PencilLine, Trash2 } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

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
import type { CharacterAnimation } from "@/model";
import { z } from "zod";

// Keep these in sync with the fixed menu dimensions below.
const CONTEXT_MENU_WIDTH = 192;
const CONTEXT_MENU_HEIGHT = 82;
const CONTEXT_MENU_VIEWPORT_PADDING = 8;
const animationLabelSchema = z.string().trim().min(1);

type ContextMenuState = {
  animation: CharacterAnimation;
  x: number;
  y: number;
};

export function useAnimationActions({
  onRename,
  onDelete,
}: {
  onRename: (animationId: string, label: string) => void;
  onDelete: (animationId: string) => void;
}) {
  const { t } = useTranslation("editor");
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [editingAnimation, setEditingAnimation] =
    useState<CharacterAnimation | null>(null);
  const [editedLabel, setEditedLabel] = useState("");

  useEffect(() => {
    if (!contextMenu) return;
    const close = () => setContextMenu(null);
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") close();
    };
    window.addEventListener("pointerdown", close);
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      window.removeEventListener("pointerdown", close);
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [contextMenu]);

  const openContextMenu = (
    event: React.MouseEvent<HTMLElement>,
    animation: CharacterAnimation,
  ) => {
    event.preventDefault();
    setContextMenu({
      animation,
      x: Math.max(
        CONTEXT_MENU_VIEWPORT_PADDING,
        Math.min(
          event.clientX,
          window.innerWidth -
            CONTEXT_MENU_WIDTH -
            CONTEXT_MENU_VIEWPORT_PADDING,
        ),
      ),
      y: Math.max(
        CONTEXT_MENU_VIEWPORT_PADDING,
        Math.min(
          event.clientY,
          window.innerHeight -
            CONTEXT_MENU_HEIGHT -
            CONTEXT_MENU_VIEWPORT_PADDING,
        ),
      ),
    });
  };

  const handleEdit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = animationLabelSchema.safeParse(editedLabel);
    if (!editingAnimation || !result.success) return;
    onRename(editingAnimation.id, result.data);
    setEditingAnimation(null);
  };

  return {
    openContextMenu,
    actions: (
      <>
        <Dialog
          open={Boolean(editingAnimation)}
          onOpenChange={(open) => !open && setEditingAnimation(null)}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("editAnimationName")}</DialogTitle>
              <DialogDescription>
                {t("renameAnimationDescription")}
              </DialogDescription>
            </DialogHeader>
            <form className="grid gap-5" onSubmit={handleEdit}>
              <label
                className="grid gap-2 text-sm font-medium"
                htmlFor="edit-animation-name"
              >
                {t("animationName")}
                <Input
                  id="edit-animation-name"
                  autoFocus
                  required
                  value={editedLabel}
                  onChange={(event) => setEditedLabel(event.target.value)}
                />
              </label>
              <DialogFooter>
                <DialogClose
                  render={<Button type="button" variant="outline" />}
                >
                  {t("cancel")}
                </DialogClose>
                <Button
                  type="submit"
                  disabled={
                    !animationLabelSchema.safeParse(editedLabel).success
                  }
                >
                  {t("saveName")}
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
            className="fixed z-50 grid w-48 gap-1 rounded-lg border bg-popover p-1.5 text-popover-foreground shadow-lg"
          >
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setEditedLabel(contextMenu.animation.label);
                setEditingAnimation(contextMenu.animation);
                setContextMenu(null);
              }}
              className="flex h-8 items-center gap-2 rounded-md px-2 text-left text-xs font-medium hover:bg-muted"
            >
              <PencilLine className="size-3.5" />
              {t("editName")}
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onDelete(contextMenu.animation.id);
                setContextMenu(null);
              }}
              className="flex h-8 items-center gap-2 rounded-md px-2 text-left text-xs font-medium text-destructive hover:bg-destructive/10"
            >
              <Trash2 className="size-3.5" />
              {t("deleteAnimation")}
            </button>
          </div>
        ) : null}
      </>
    ),
  };
}
