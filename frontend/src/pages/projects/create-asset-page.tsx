import { ArrowLeft } from "lucide-react";
import { Link, useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

import { CreateAssetForm } from "@/features/generation";
import { assetKindSchema } from "@/model/asset";
import { useEnqueueGenerationMutation } from "@/model/generation";
import {
  useCreateProjectTagMutation,
  useProjectListQuery,
  useProjectTagsQuery,
  useUpdateProjectTagMutation,
} from "@/model/project";

export function CreateAssetPage({
  projectId,
  rawKind,
}: {
  projectId: string;
  rawKind: string;
}) {
  const { t } = useTranslation(["generation", "common", "projects"]);
  const navigate = useNavigate();
  const { data: projects = [] } = useProjectListQuery();
  const { data: projectTags = [] } = useProjectTagsQuery(projectId);
  const createTagMutation = useCreateProjectTagMutation();
  const updateTagMutation = useUpdateProjectTagMutation();
  const {
    error: enqueueError,
    isPending: isEnqueuePending,
    mutateAsync: enqueueGeneration,
    reset: resetEnqueue,
  } = useEnqueueGenerationMutation();
  const project = projects.find((item) => item.id === projectId);
  const kindResult = assetKindSchema.safeParse(rawKind);
  const kind = kindResult.success ? kindResult.data : undefined;
  const availableTags = projectTags;

  if (!project || !kind) return null;

  const goBack = () =>
    void navigate({ to: "/projects/$projectId", params: { projectId } });

  return (
    <div className="relative min-h-screen bg-background text-foreground">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="absolute top-6 left-6 inline-flex items-center gap-2 text-base text-muted-foreground transition-colors hover:text-foreground sm:top-7 sm:left-8"
      >
        <ArrowLeft className="size-5" />
        {t("projects:assetLibrary")}
      </Link>
      <main className="mx-auto w-full px-5 pt-24 pb-16 sm:px-8 sm:pt-28 lg:w-[70vw] lg:max-w-[90rem] lg:pb-24">
        <header>
          <h1 className="text-5xl leading-tight font-normal sm:text-6xl">
            {t("pageTitle", { kind: t(`common:assetKinds.${kind}`) })}
          </h1>
          <p className="mt-4 text-lg text-muted-foreground sm:text-xl">
            {t("pageDescription")}
          </p>
        </header>
        <section className="mt-12 border bg-background p-7 shadow-sm sm:p-10 lg:mt-14 lg:p-10">
          <CreateAssetForm
            availableTags={availableTags}
            kind={kind}
            onCancel={goBack}
            error={enqueueError}
            isSubmitting={isEnqueuePending}
            onCreateTag={(tag) =>
              createTagMutation.mutateAsync({ projectId, tag })
            }
            onUpdateTag={(currentTag, tag) => {
              if (!currentTag.tagId) return tag;
              return updateTagMutation.mutateAsync({
                projectId,
                tagId: currentTag.tagId,
                tag,
              });
            }}
            onCreate={async (request) => {
              resetEnqueue();
              try {
                await enqueueGeneration({ projectId: project.id, request });
                goBack();
              } catch {
                // Keep the form mounted so the user can retry the request.
              }
            }}
          />
        </section>
      </main>
    </div>
  );
}
