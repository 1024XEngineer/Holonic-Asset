import { AlertTriangle, LoaderCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useRecordQuery } from "@/model";

import { EditorWorkspace } from "./editor-workspace";

export function AssetEditor({
  projectId,
  assetId,
  onExitEditor,
}: {
  projectId: string;
  assetId: string;
  onExitEditor: () => void;
}) {
  const recordQuery = useRecordQuery(projectId, assetId);

  if (recordQuery.isPending) {
    return (
      <EditorStatus
        icon={<LoaderCircle className="size-5 animate-spin" />}
        title="Loading asset editor"
        description="Preparing the latest asset record."
      />
    );
  }

  if (recordQuery.error) {
    return (
      <EditorStatus
        icon={<AlertTriangle className="size-5" />}
        title="Unable to open asset"
        description={recordQuery.error.message}
        action={
          <>
            <Button
              variant="outline"
              onClick={() => void recordQuery.refetch()}
            >
              Try again
            </Button>
            <Button onClick={onExitEditor}>Back to library</Button>
          </>
        }
      />
    );
  }

  if (
    recordQuery.data.record.mode !== "character" &&
    recordQuery.data.record.mode !== "object"
  ) {
    return (
      <EditorStatus
        icon={<AlertTriangle className="size-5" />}
        title="Editor unavailable"
        description="This editor currently supports character and object assets."
        action={<Button onClick={onExitEditor}>Back to library</Button>}
      />
    );
  }

  return <EditorWorkspace data={recordQuery.data} onBack={onExitEditor} />;
}

function EditorStatus({
  icon,
  title,
  description,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <main className="grid min-h-dvh place-items-center bg-muted/30 p-6">
      <section className="w-full max-w-md border bg-background p-6 shadow-sm">
        <div className="grid size-10 place-items-center border bg-muted text-muted-foreground">
          {icon}
        </div>
        <h1 className="mt-5 text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          {description}
        </p>
        {action ? <div className="mt-6 flex gap-2">{action}</div> : null}
      </section>
    </main>
  );
}
