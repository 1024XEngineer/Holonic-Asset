import { useState } from "react";
import { ArrowLeft, ArrowRight, LoaderCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { DropdownField } from "@/components/ui/custom/dropdown-field";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { isPerspective } from "@/model/project";
import { projectContextOptions } from "../lib/project-context";
import type { NewProjectController } from "./use-new-project-controller";

export function GuidedProjectFlow({
  project,
}: {
  project: NewProjectController;
}) {
  const { form, preview } = project;
  const { instance: newProjectForm, isGenerating, step } = form;
  const [isImagePreviewOpen, setIsImagePreviewOpen] = useState(false);

  return (
    <form
      className="mx-auto grid w-full max-w-2xl gap-6"
      onSubmit={(event) => {
        event.preventDefault();
        if (step === 2) void newProjectForm.handleSubmit();
        else form.next();
      }}
    >
      {step === 2 ? (
        <div>
          <h2 className="text-lg font-semibold">Project overview</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Upload an image that represents the game world, characters, and
            visual language together.
          </p>
        </div>
      ) : null}
      {step === 1 ? (
        <>
          <newProjectForm.Field name="name">
            {(field) => (
              <label className="grid gap-2 text-sm font-semibold">
                Project name
                <input
                  autoFocus
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
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
          <newProjectForm.Field name="perspective">
            {(field) => (
              <DropdownField
                label="Perspective"
                value={field.state.value}
                options={projectContextOptions.perspectives}
                onChange={(value) => {
                  if (isPerspective(value)) field.handleChange(value);
                }}
              />
            )}
          </newProjectForm.Field>
          <newProjectForm.Field name="description">
            {(field) => (
              <label className="grid gap-2 text-sm font-semibold">
                Game description
                <textarea
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
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
              {preview.isGenerating ? (
                <div className="relative grid size-full place-items-center overflow-hidden">
                  <img
                    src="/project/reference/reference.png"
                    alt=""
                    aria-hidden
                    className="absolute inset-0 size-full scale-105 object-cover opacity-45 blur-md"
                  />
                  <div className="absolute inset-0 bg-background/25" />
                  <LoaderCircle className="relative size-8 animate-spin text-foreground" />
                </div>
              ) : preview.url ? (
                <button
                  type="button"
                  className="size-full cursor-zoom-in"
                  aria-label="Preview generated project overview"
                  onClick={() => setIsImagePreviewOpen(true)}
                >
                  <img
                    src={preview.url}
                    alt="Generated project overview"
                    className="size-full object-cover"
                  />
                </button>
              ) : null}
            </div>
            <Button
              type="button"
              variant="outline"
              className="w-full"
              onClick={preview.generate}
              disabled={preview.isGenerating}
            >
              {preview.url ? "Regenerate preview" : "Generate preview"}
            </Button>
            {preview.error ? (
              <p role="alert" className="text-sm text-destructive">
                {preview.error}
              </p>
            ) : null}
          </TabsContent>
          <TabsContent value="upload">
            <ImageDropzone
              className="aspect-[16/9] min-h-0"
              label="Upload project overview image"
              value={preview.url || undefined}
              error={preview.error}
              onChange={(file) => {
                if (file) preview.setFile(file);
                else preview.clear();
              }}
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
          className="inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
          type="submit"
          disabled={step === 2 && isGenerating}
        >
          {step === 2 ? "Submit" : "Next"}
          <ArrowRight size={16} />
        </button>
      </div>
      <Dialog open={isImagePreviewOpen} onOpenChange={setIsImagePreviewOpen}>
        <DialogContent className="max-w-5xl p-2 sm:max-w-5xl">
          <DialogTitle className="sr-only">
            Generated project overview
          </DialogTitle>
          {preview.url ? (
            <img
              src={preview.url}
              alt="Generated project overview"
              className="max-h-[80vh] w-full object-contain"
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </form>
  );
}
