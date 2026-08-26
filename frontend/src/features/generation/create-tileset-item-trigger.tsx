import { LoaderCircle, PackagePlus } from "lucide-react";
import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

import { ItemShapePicker } from "./create-asset/item-shape-picker";
import type { CreateTilesetItemRequest } from "./types";

const requestSchema = z.object({
  itemName: z.string().trim().min(1),
  itemDescription: z.string().trim().min(1),
  shape: z.array(z.tuple([z.number().int(), z.number().int()])).min(1),
  creativeBrief: z.string().trim().min(1),
});

const defaultRequest: CreateTilesetItemRequest = {
  itemName: "",
  itemDescription: "",
  shape: [[0, 0]],
  creativeBrief: "",
};

export function CreateTilesetItemTrigger({
  children,
  isGenerating,
  onGenerate,
}: {
  children: (openDialog: () => void) => React.ReactNode;
  isGenerating: boolean;
  onGenerate: (request: CreateTilesetItemRequest) => void;
}) {
  const { t } = useTranslation(["generation", "editor"]);
  const [open, setOpen] = useState(false);
  const [request, setRequest] = useState(defaultRequest);

  const openDialog = () => {
    setRequest(defaultRequest);
    setOpen(true);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = requestSchema.safeParse(request);
    if (!result.success || isGenerating) return;

    onGenerate(result.data);
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {children(openDialog)}
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span className="grid size-8 place-items-center rounded-lg bg-primary/10 text-primary">
              <PackagePlus className="size-4" />
            </span>
            {t("addTilesetItem")}
          </DialogTitle>
          <DialogDescription>{t("tilesetItemDescription")}</DialogDescription>
        </DialogHeader>
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <label className="grid gap-2 text-sm font-medium" htmlFor="item-name">
            {t("itemName", { number: 1 })}
            <Input
              id="item-name"
              autoFocus
              required
              placeholder={t("itemNamePlaceholder")}
              value={request.itemName}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  itemName: event.target.value,
                }))
              }
            />
          </label>
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="item-description"
          >
            {t("itemDescription", { number: 1 })}
            <Textarea
              id="item-description"
              required
              className="min-h-20 resize-y"
              value={request.itemDescription}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  itemDescription: event.target.value,
                }))
              }
            />
          </label>
          <ItemShapePicker
            shape={request.shape}
            onChange={(shape) =>
              setRequest((current) => ({ ...current, shape }))
            }
          />
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="new-item-prompt"
          >
            {t("creativeBrief")}
            <Textarea
              id="new-item-prompt"
              required
              className="min-h-24 resize-y"
              placeholder={t("tilesetItemPromptPlaceholder")}
              value={request.creativeBrief}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  creativeBrief: event.target.value,
                }))
              }
            />
          </label>
          <Button
            type="submit"
            disabled={!requestSchema.safeParse(request).success || isGenerating}
          >
            {isGenerating ? (
              <LoaderCircle className="animate-spin" />
            ) : (
              <PackagePlus />
            )}
            {isGenerating ? t("queueingTilesetItem") : t("addTilesetItem")}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}
