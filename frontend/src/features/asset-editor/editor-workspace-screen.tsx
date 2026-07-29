import { Button } from "@/components/ui/button";
import type { EditorWorkspaceData } from "@/model";

import { EditorWorkspace } from "./editor-workspace";

export function EditorWorkspaceScreen({
  data,
  error,
  isLoading,
  onBack,
  onRetry,
}: {
  data?: EditorWorkspaceData;
  error: Error | null;
  isLoading: boolean;
  onBack: () => void;
  onRetry: () => void;
}) {
  if (isLoading) {
    return (
      <EditorRouteState
        description="Loading the asset editor workspace."
        title="Preparing asset studio"
      />
    );
  }

  if (error || !data) {
    return (
      <EditorRouteState
        description={error?.message || "The requested asset was not found."}
        title="Unable to open asset"
        actions={
          <>
            <Button variant="outline" onClick={onRetry}>
              Try again
            </Button>
            <Button onClick={onBack}>Back to project</Button>
          </>
        }
      />
    );
  }

  return <EditorWorkspace data={data} onBack={onBack} />;
}

function EditorRouteState({
  actions,
  description,
  title,
}: {
  actions?: React.ReactNode;
  description: string;
  title: string;
}) {
  return (
    <main className="grid h-screen place-items-center bg-[#f7f5f0] px-6 text-[#2d2923]">
      <section className="w-full max-w-md rounded-2xl border border-black/10 bg-white p-6 text-center shadow-sm">
        <h1 className="font-serif text-2xl">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-[#71685d]">{description}</p>
        {actions ? (
          <div className="mt-5 flex justify-center gap-2">{actions}</div>
        ) : null}
      </section>
    </main>
  );
}
