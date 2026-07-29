import { Sparkles } from "lucide-react";
import { useState } from "react";

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
import type { GenerateAnimationRequest } from "@/model";

type CreateAnimationDialogProps = {
  children: (openDialog: () => void) => React.ReactNode;
  isGenerating: boolean;
  onGenerate: (request: GenerateAnimationRequest) => void;
};

export function CreateAnimationDialog({
  children,
  isGenerating,
  onGenerate,
}: CreateAnimationDialogProps) {
  const [open, setOpen] = useState(false);
  const [animationName, setAnimationName] = useState("");
  const [generationPrompt, setGenerationPrompt] = useState("");

  const openDialog = () => {
    setAnimationName("");
    setGenerationPrompt("");
    setOpen(true);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const label = animationName.trim();
    const prompt = generationPrompt.trim();

    if (!label || !prompt || isGenerating) return;

    onGenerate({ label, prompt });
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {children(openDialog)}
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generate animation</DialogTitle>
          <DialogDescription>
            Describe a motion and generation will add a playable clip to this
            character.
          </DialogDescription>
        </DialogHeader>
        <form className="grid gap-5" onSubmit={handleSubmit}>
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="animation-name"
          >
            Animation name
            <Input
              id="animation-name"
              autoFocus
              required
              placeholder="e.g. Cast spell"
              value={animationName}
              onChange={(event) => setAnimationName(event.target.value)}
            />
          </label>
          <label
            className="grid gap-2 text-sm font-medium"
            htmlFor="generated-animation-prompt"
          >
            Generation prompt
            <Textarea
              id="generated-animation-prompt"
              required
              className="min-h-28 resize-y"
              placeholder="Describe the motion, timing, and pose..."
              value={generationPrompt}
              onChange={(event) => setGenerationPrompt(event.target.value)}
            />
          </label>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>
              Cancel
            </DialogClose>
            <Button
              type="submit"
              disabled={
                isGenerating ||
                !animationName.trim() ||
                !generationPrompt.trim()
              }
            >
              <Sparkles />
              Generate animation
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
