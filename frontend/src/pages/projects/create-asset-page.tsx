import { ArrowLeft } from "lucide-react";
import { Link, useNavigate } from "@tanstack/react-router";

import { getAssetKindConfig } from "@/components/asset-kind";
import { CreateAssetForm } from "@/features/generation/create-asset-form";
import { assetKindSchema } from "@/model/asset";
import { useEnqueueGenerationMutation } from "@/model/generation";
import { useProjectListQuery } from "@/model/project";

export function CreateAssetPage({
  projectId,
  rawKind,
}: {
  projectId: string;
  rawKind: string;
}) {
  const navigate = useNavigate();
  const { data: projects = [] } = useProjectListQuery();
  const {
    error: enqueueError,
    isPending: isEnqueuePending,
    mutateAsync: enqueueGeneration,
    reset: resetEnqueue,
  } = useEnqueueGenerationMutation();
  const project = projects.find((item) => item.id === projectId);
  const kindResult = assetKindSchema.safeParse(rawKind);
  const kind = kindResult.success ? kindResult.data : undefined;

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
        Asset library
      </Link>
      <main className="mx-auto w-full px-5 pt-24 pb-16 sm:px-8 sm:pt-28 lg:w-[70vw] lg:max-w-[90rem] lg:pb-24">
        <header>
          <h1 className="text-5xl leading-tight font-normal sm:text-6xl">
            Create {getAssetKindConfig(kind).label}
          </h1>
          <p className="mt-4 text-lg text-muted-foreground sm:text-xl">
            Set the production details and generation context for this new
            asset.
          </p>
        </header>
        <section className="mt-12 border bg-background p-7 shadow-sm sm:p-10 lg:mt-14 lg:p-10">
          <CreateAssetForm
            kind={kind}
            project={project}
            onCancel={goBack}
            error={enqueueError}
            isSubmitting={isEnqueuePending}
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
