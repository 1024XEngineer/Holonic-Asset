import { useRef } from "react";
import {
  ArrowLeft,
  ArrowRight,
  FilePlus2,
  Gamepad2,
  Lightbulb,
  Link2,
  Upload,
} from "lucide-react";

import { DropdownField } from "@/components/ui/custom/dropdown-field";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { NewProjectController } from "./state/use-new-project-controller";
import { projectContextOptions } from "../project-context";

export function NewProjectWorkspace({
  project,
}: {
  project: NewProjectController;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { backToLibrary, existingGameImport, form, preview, start } = project;
  const { instance: newProjectForm, selectedStart, step } = form;

  return (
    <main className="relative w-full">
      <button
        type="button"
        className="absolute left-4 top-4 inline-flex whitespace-nowrap items-center gap-2 bg-transparent text-base font-semibold text-muted-foreground hover:text-foreground sm:left-6 sm:top-6"
        onClick={selectedStart ? form.returnToStart : backToLibrary}
      >
        <ArrowLeft size={16} />
        {selectedStart ? "Choose a starting point" : "Project library"}
      </button>
      <div className="mx-auto w-full max-w-6xl px-4 py-8 pb-20 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <p className="mt-16 font-mono text-sm font-bold uppercase tracking-[0.12em] text-muted-foreground">
            New project
          </p>
          <h1 className="mt-3 text-4xl font-semibold leading-tight">
            {selectedStart
              ? selectedStart === "blank"
                ? "Start with as little as you like"
                : "Tell us about your game"
              : "Where would you like to start?"}
          </h1>
          <p className="mt-4 text-base leading-7 text-muted-foreground">
            {selectedStart
              ? selectedStart === "blank"
                ? "Give the workspace a name. You can add context later."
                : "Add the project basics and describe your idea. We will use them to guide your first asset generation."
              : "Pick the amount of structure you need. You can always add more project context later."}
          </p>
        </div>
        {!selectedStart ? (
          <div className="mx-auto mt-10 grid max-w-6xl gap-4 md:grid-cols-3">
            <button
              type="button"
              className="flex min-h-52 flex-col rounded-md border bg-card p-6 text-left shadow-sm transition-[transform,border-color,box-shadow] hover:-translate-y-1 hover:border-foreground hover:shadow-xl focus-visible:outline-3 focus-visible:outline-ring focus-visible:outline-offset-3"
              onClick={start.openExistingGameImport}
            >
              <span className="grid size-10 place-items-center rounded-md bg-muted text-foreground">
                <Gamepad2 size={20} />
              </span>
              <h2 className="mt-auto text-base font-semibold">Existing game</h2>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">
                Import a playable link or upload a local build so we can learn
                its direction.
              </p>
            </button>
            <button
              type="button"
              className="flex min-h-52 flex-col rounded-md border bg-card p-6 text-left shadow-sm transition-[transform,border-color,box-shadow] hover:-translate-y-1 hover:border-foreground hover:shadow-xl focus-visible:outline-3 focus-visible:outline-ring focus-visible:outline-offset-3"
              onClick={start.chooseIdea}
            >
              <span className="grid size-10 place-items-center rounded-md bg-muted text-foreground">
                <Lightbulb size={20} />
              </span>
              <h2 className="mt-auto text-base font-semibold">
                I have an idea
              </h2>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">
                Describe the game, generate a visual direction, and refine it
                until it feels right.
              </p>
            </button>
            <button
              type="button"
              className="flex min-h-52 flex-col rounded-md border bg-card p-6 text-left shadow-sm transition-[transform,border-color,box-shadow] hover:-translate-y-1 hover:border-foreground hover:shadow-xl focus-visible:outline-3 focus-visible:outline-ring focus-visible:outline-offset-3"
              onClick={start.chooseBlank}
            >
              <span className="grid size-10 place-items-center rounded-md bg-muted text-foreground">
                <FilePlus2 size={20} />
              </span>
              <h2 className="mt-auto text-base font-semibold">Blank project</h2>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">
                Open a flexible workspace. Add context if useful, or create an
                asset immediately.
              </p>
            </button>
          </div>
        ) : (
          <form
            className="mx-auto mt-12 grid max-w-2xl gap-6"
            onSubmit={(event) => {
              event.preventDefault();
              if (selectedStart === "blank" || step === 2)
                void newProjectForm.handleSubmit();
              else form.next();
            }}
          >
            <div>
              <h2 className="text-lg font-semibold">
                {selectedStart === "blank"
                  ? "Name your project"
                  : step === 1
                    ? "Project context"
                    : "Project overview"}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {selectedStart === "blank"
                  ? "You can add more project context later."
                  : step === 1
                    ? "These defaults guide the visual direction of your project."
                    : "Upload an image that represents the game world, characters, and visual language together."}
              </p>
            </div>
            {selectedStart === "blank" ? (
              <newProjectForm.Field name="name">
                {(field) => (
                  <label className="grid gap-2 text-sm font-semibold">
                    Project name
                    <input
                      autoFocus
                      value={field.state.value}
                      onChange={(event) =>
                        field.handleChange(event.target.value)
                      }
                      className="w-full rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
                      placeholder="e.g. Moonlit Orchard"
                    />
                  </label>
                )}
              </newProjectForm.Field>
            ) : step === 1 ? (
              <>
                <newProjectForm.Field name="name">
                  {(field) => (
                    <label className="grid gap-2 text-sm font-semibold">
                      Project name
                      <input
                        autoFocus
                        value={field.state.value}
                        onChange={(event) =>
                          field.handleChange(event.target.value)
                        }
                        className="w-full rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
                        placeholder="e.g. Moonlit Orchard"
                      />
                    </label>
                  )}
                </newProjectForm.Field>
                <div className="grid gap-5 sm:grid-cols-2">
                  <newProjectForm.Field name="gameType">
                    {(field) => (
                      <DropdownField
                        label="Game type"
                        value={field.state.value}
                        options={projectContextOptions.gameTypes}
                        onChange={field.handleChange}
                      />
                    )}
                  </newProjectForm.Field>
                  <newProjectForm.Field name="platform">
                    {(field) => (
                      <DropdownField
                        label="Target platform"
                        value={field.state.value}
                        options={projectContextOptions.platforms}
                        onChange={field.handleChange}
                      />
                    )}
                  </newProjectForm.Field>
                </div>
                <newProjectForm.Field name="visualStyle">
                  {(field) => (
                    <label className="grid gap-2 text-sm font-semibold">
                      Visual style
                      <input
                        value={field.state.value}
                        onChange={(event) =>
                          field.handleChange(event.target.value)
                        }
                        className="w-full rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
                        placeholder="Pixel art, hand-painted, cartoon..."
                      />
                    </label>
                  )}
                </newProjectForm.Field>
                <newProjectForm.Field name="description">
                  {(field) => (
                    <label className="grid gap-2 text-sm font-semibold">
                      Game description
                      <textarea
                        value={field.state.value}
                        onChange={(event) =>
                          field.handleChange(event.target.value)
                        }
                        className="min-h-28 w-full resize-none rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
                        placeholder="What is the player doing? What should the world feel like?"
                      />
                    </label>
                  )}
                </newProjectForm.Field>
              </>
            ) : (
              <Tabs
                value={preview.mode}
                onValueChange={(value) => {
                  if (value === "generate") preview.selectGenerate();
                  else preview.selectUpload();
                }}
              >
                <TabsList
                  aria-label="Project overview source"
                  className="grid w-full grid-cols-2"
                >
                  <TabsTrigger value="generate">Generate</TabsTrigger>
                  <TabsTrigger value="upload">Upload</TabsTrigger>
                </TabsList>
                <TabsContent value="generate" className="grid gap-3">
                  <div className="aspect-[16/9] overflow-hidden rounded-md border bg-muted/30">
                    {preview.url ? (
                      <img
                        src={preview.url}
                        alt="Generated project overview"
                        className="size-full object-cover"
                      />
                    ) : null}
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
                    onClick={preview.generate}
                  >
                    {preview.url ? "Regenerate preview" : "Generate preview"}
                  </Button>
                </TabsContent>
                <TabsContent value="upload">
                  <ImageDropzone
                    className="aspect-[16/9] min-h-0"
                    label="Upload project overview image"
                    previewUrl={preview.url || undefined}
                    onSelect={preview.setFile}
                    onClear={preview.clear}
                  />
                </TabsContent>
              </Tabs>
            )}
            <div className="mt-2 flex justify-between border-t pt-6">
              <button
                type="button"
                className="inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
                onClick={form.previous}
              >
                <ArrowLeft size={16} /> Previous
              </button>
              <button
                className={
                  "inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
                }
                type="submit"
              >
                {selectedStart === "blank" || step === 2 ? "Submit" : "Next"}
                <ArrowRight size={16} />
              </button>
            </div>
          </form>
        )}
      </div>
      <Dialog
        open={existingGameImport.isOpen}
        onOpenChange={(open) => !open && existingGameImport.dismiss()}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Import an existing game</DialogTitle>
            <DialogDescription>
              Share a playable link or upload a local game build to begin its
              project setup.
            </DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-2 rounded-lg bg-muted p-1">
            <Button
              variant={existingGameImport.mode === "link" ? "default" : "ghost"}
              onClick={existingGameImport.selectLink}
            >
              <Link2 /> Game link
            </Button>
            <Button
              variant={existingGameImport.mode === "file" ? "default" : "ghost"}
              onClick={existingGameImport.selectFile}
            >
              <Upload /> Local files
            </Button>
          </div>
          {existingGameImport.mode === "link" ? (
            <label className="grid gap-2 text-sm font-medium">
              Playable URL
              <Input
                type="url"
                placeholder="https://your-game.example"
                value={existingGameImport.gameUrl}
                onChange={(event) =>
                  existingGameImport.setGameUrl(event.target.value)
                }
              />
            </label>
          ) : (
            <>
              <input
                ref={fileInputRef}
                className="sr-only"
                type="file"
                accept=".zip,.html,.exe,.dmg,.apk"
                onChange={(event) =>
                  existingGameImport.setGameFile(
                    event.target.files?.[0] ?? null,
                  )
                }
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="grid min-h-32 place-items-center rounded-xl border border-dashed p-5 text-center text-sm hover:bg-muted/50"
              >
                <span>
                  <Upload className="mx-auto mb-2 size-5 text-muted-foreground" />
                  {existingGameImport.gameFile?.name ?? "Choose a game build"}
                </span>
              </button>
            </>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={existingGameImport.dismiss}>
              Cancel
            </Button>
            <Button
              disabled={
                existingGameImport.mode === "link"
                  ? !existingGameImport.gameUrl.trim()
                  : !existingGameImport.gameFile
              }
              onClick={existingGameImport.continue}
            >
              Continue
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
  );
}
