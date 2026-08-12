import { ArrowLeft, ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
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
  const { t } = useTranslation("projects");
  const { form, preview } = project;
  const { instance: newProjectForm, step } = form;

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
          <h2 className="text-lg font-semibold">{t("projectOverview")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("overviewDescription")}
          </p>
        </div>
      ) : null}
      {step === 1 ? (
        <>
          <newProjectForm.Field name="name">
            {(field) => (
              <label className="grid gap-2 text-sm font-semibold">
                {t("projectName")}
                <input
                  autoFocus
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  className="w-full rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
                  placeholder={t("projectNamePlaceholder")}
                />
              </label>
            )}
          </newProjectForm.Field>
          <div className="grid gap-5 sm:grid-cols-2">
            <newProjectForm.Field name="gameType">
              {(field) => (
                <DropdownField
                  label={t("gameType")}
                  value={field.state.value}
                  options={projectContextOptions.gameTypes}
                  onChange={field.handleChange}
                />
              )}
            </newProjectForm.Field>
            <newProjectForm.Field name="platform">
              {(field) => (
                <DropdownField
                  label={t("platform")}
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
                label={t("perspective")}
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
                {t("gameDescription")}
                <textarea
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  className="min-h-28 w-full resize-none rounded-md border bg-background px-3 py-2.5 font-normal outline-none focus:border-ring focus:ring-3 focus:ring-ring/25"
                  placeholder={t("gameDescriptionPlaceholder")}
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
            aria-label={t("overviewSource")}
            className="grid w-full grid-cols-2"
          >
            <TabsTrigger value="generate">{t("generate")}</TabsTrigger>
            <TabsTrigger value="upload">{t("upload")}</TabsTrigger>
          </TabsList>
          <TabsContent value="generate" className="grid gap-3">
            <div className="aspect-[16/9] overflow-hidden rounded-md border bg-muted/30">
              {preview.url ? (
                <img
                  src={preview.url}
                  alt={t("generatedOverview")}
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
              {preview.url ? t("regeneratePreview") : t("generatePreview")}
            </Button>
          </TabsContent>
          <TabsContent value="upload">
            <ImageDropzone
              className="aspect-[16/9] min-h-0"
              label={t("uploadOverview")}
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
          <ArrowLeft size={16} /> {t("previous")}
        </button>
        <button
          className="inline-flex items-center justify-center gap-2 rounded-md px-3.5 py-2.5 text-sm font-semibold hover:bg-muted"
          type="submit"
        >
          {step === 2 ? t("submit") : t("next")}
          <ArrowRight size={16} />
        </button>
      </div>
    </form>
  );
}
