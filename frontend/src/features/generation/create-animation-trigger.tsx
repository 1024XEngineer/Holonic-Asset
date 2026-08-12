import { Sparkles } from "lucide-react";
import { useState, type FormEvent } from "react";
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
import { Textarea } from "@/components/ui/textarea";
import {
  generateAnimationRequestSchema,
  type GenerateAnimationRequest,
} from "@/model";

type CreateAnimationTriggerProps = {
  children: (openDialog: () => void) => React.ReactNode;
  isGenerating: boolean;
  onGenerate: (request: GenerateAnimationRequest) => void;
};

export function CreateAnimationTrigger({
  children,
  isGenerating,
  onGenerate,
}: CreateAnimationTriggerProps) {
  const { t } = useTranslation(["generation", "common"]);
  const [open, setOpen] = useState(false);
  const [animationName, setAnimationName] = useState("");
  const [generationPrompt, setGenerationPrompt] = useState("");

  const openDialog = () => {
    setAnimationName("");
    setGenerationPrompt("");
    setOpen(true);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const result = generateAnimationRequestSchema.safeParse({
      label: animationName,
      prompt: generationPrompt,
    });
    if (!result.success || isGenerating) return;

    onGenerate(result.data);
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {children(openDialog)}
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("generateAnimation")}</DialogTitle>
          <DialogDescription>{t("animationDescription")}</DialogDescription>
        </DialogHeader>
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="animation-name"
          >
            {t("animationName")}
            <Input
              id="animation-name"
              autoFocus
              required
              placeholder={t("castSpellPlaceholder")}
              value={animationName}
              onChange={(event) => setAnimationName(event.target.value)}
            />
          </label>
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="generated-animation-prompt"
          >
            {t("generationPrompt")}
            <Textarea
              id="generated-animation-prompt"
              required
              className="min-h-28 resize-y"
              placeholder={t("motionPlaceholder")}
              value={generationPrompt}
              onChange={(event) => setGenerationPrompt(event.target.value)}
            />
          </label>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>
              {t("common:actions.cancel")}
            </DialogClose>
            <Button
              type="submit"
              disabled={
                isGenerating ||
                !generateAnimationRequestSchema.safeParse({
                  label: animationName,
                  prompt: generationPrompt,
                }).success
              }
            >
              <Sparkles />
              {t("generateAnimation")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
