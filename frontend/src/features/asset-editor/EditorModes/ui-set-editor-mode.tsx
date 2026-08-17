import {
  ArrowUp,
  Badge,
  Circle,
  Gauge,
  ImagePlus,
  MousePointer2,
  PanelsTopLeft,
  SlidersHorizontal,
  TextCursorInput,
  ToggleRight,
  Type,
  RotateCcw,
  X,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { AssetWorkspaceData, UISetComponent } from "@/model";

import { UISetCanvas } from "../Canvas/UISetCanvas";
import { EditorHeader } from "../Header/editor-header";
import { InspectorHistory } from "../Inspector/history";
import { useEditorSession } from "../state";

const componentIcons = {
  panel: PanelsTopLeft,
  label: Type,
  button: MousePointer2,
  input: TextCursorInput,
  badge: Badge,
  progress: Gauge,
  toggle: ToggleRight,
  icon: Circle,
  slider: SlidersHorizontal,
} satisfies Record<UISetComponent["kind"], typeof PanelsTopLeft>;

export function UISetEditorMode({
  data,
  onBack,
}: {
  data: AssetWorkspaceData;
  onBack: () => void;
}) {
  const { t } = useTranslation("editor");
  const sessionIdentity = `${data.asset.projectId}\0${data.asset.id}`;
  const initialRecordRef = useRef({
    identity: sessionIdentity,
    record: structuredClone(data.record),
  });
  if (initialRecordRef.current.identity !== sessionIdentity) {
    initialRecordRef.current = {
      identity: sessionIdentity,
      record: structuredClone(data.record),
    };
  }
  const session = useEditorSession({
    target: { projectId: data.asset.projectId, assetId: data.asset.id },
    initialRecord: data.record,
  });
  const { snapshot } = session;
  const [selectedComponentId, setSelectedComponentId] = useState<string | null>(
    null,
  );
  useEffect(() => {
    session.syncExternalRecord(data.record);
  }, [data.record, session.syncExternalRecord]);

  const components =
    snapshot.record.mode === "uiset" ? snapshot.record.uiset.components : [];
  const selectedComponent = components.find(
    (component) => component.id === selectedComponentId,
  );
  const initialComponents =
    initialRecordRef.current.record.mode === "uiset"
      ? initialRecordRef.current.record.uiset.components
      : [];
  const initialSelectedComponent = initialComponents.find(
    (component) => component.id === selectedComponentId,
  );
  useEffect(() => {
    if (
      selectedComponentId &&
      !components.some((component) => component.id === selectedComponentId)
    ) {
      setSelectedComponentId(null);
    }
  }, [components, selectedComponentId]);

  const status =
    snapshot.saveState.phase === "saving"
      ? t("savingChanges")
      : snapshot.saveState.phase === "failed"
        ? snapshot.saveState.message
        : snapshot.dirty
          ? t("unsavedChanges")
          : t("allChangesSaved");

  return (
    <>
      <EditorHeader
        assetKind="uiset"
        assetName={data.asset.name}
        version={data.asset.version}
        projectName={data.projectName}
        status={status}
        canUndo={snapshot.canUndo}
        canRedo={snapshot.canRedo}
        isDirty={snapshot.dirty}
        isSaving={snapshot.saveState.phase === "saving"}
        generationTasks={[]}
        onBack={onBack}
        onUndo={() => session.dispatch({ type: "history.undo" })}
        onRedo={() => session.dispatch({ type: "history.redo" })}
        onSave={() => void session.save()}
      />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        <UISetComponentList
          components={components}
          selectedComponentId={selectedComponentId}
          onSelect={setSelectedComponentId}
        />
        <UISetCanvas
          model={{ components, selectedComponentId }}
          onEvent={({ componentId }) => setSelectedComponentId(componentId)}
        />
        <UISetInspector
          component={selectedComponent}
          prompt={snapshot.record.prompt}
          history={data.asset.history}
          onClearSelection={() => setSelectedComponentId(null)}
          onComponentLabelChange={(value) => {
            if (!selectedComponent) return;
            session.dispatch({
              type: "uiset.component.label.set",
              componentId: selectedComponent.id,
              label: value,
            });
          }}
          onRestoreComponent={() => {
            if (!initialSelectedComponent) return;
            session.dispatch({
              type: "uiset.component.restore",
              component: initialSelectedComponent,
            });
          }}
          onPromptChange={(value) =>
            session.dispatch({ type: "prompt.set", value })
          }
        />
      </div>
    </>
  );
}

function UISetComponentList({
  components,
  selectedComponentId,
  onSelect,
}: {
  components: UISetComponent[];
  selectedComponentId: string | null;
  onSelect: (componentId: string) => void;
}) {
  const { t } = useTranslation("editor");
  return (
    <aside className="flex w-full shrink-0 flex-col border-b bg-background lg:h-full lg:w-64 lg:border-r lg:border-b-0">
      <div className="border-b px-4 py-3">
        <div>
          <p className="text-xs font-semibold uppercase text-muted-foreground">
            {t("uiSetComponents")}
          </p>
        </div>
      </div>
      <ScrollArea className="max-h-56 flex-1 lg:max-h-none">
        <div className="space-y-1 p-3">
          {components.map((component) => {
            const Icon = componentIcons[component.kind];
            const selected = component.id === selectedComponentId;
            return (
              <button
                key={component.id}
                type="button"
                aria-pressed={selected}
                onClick={() => onSelect(component.id)}
                className={`flex w-full items-center gap-2 rounded-md px-2 py-2 text-left transition-colors ${selected ? "bg-primary/10 text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
              >
                <Icon className="size-4 shrink-0 text-primary" />
                <span className="min-w-0 flex-1 truncate text-xs font-medium">
                  {component.label}
                </span>
                <span className="text-[10px] uppercase text-muted-foreground">
                  {component.kind}
                </span>
              </button>
            );
          })}
          {components.length === 0 ? (
            <p className="px-2 py-6 text-center text-xs text-muted-foreground">
              {t("noUISetComponents")}
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </aside>
  );
}

function UISetInspector({
  component,
  prompt,
  history,
  onClearSelection,
  onComponentLabelChange,
  onRestoreComponent,
  onPromptChange,
}: {
  component: UISetComponent | undefined;
  prompt: string;
  history: AssetWorkspaceData["asset"]["history"];
  onClearSelection: () => void;
  onComponentLabelChange: (value: string) => void;
  onRestoreComponent: () => void;
  onPromptChange: (value: string) => void;
}) {
  const { t } = useTranslation("editor");
  return (
    <aside className="flex min-h-0 w-full shrink-0 flex-col border-t bg-background lg:w-80 lg:border-t-0 lg:border-l">
      <Tabs defaultValue="edit" className="flex min-h-0 flex-1 flex-col gap-0">
        <div className="border-b px-3 py-2">
          <TabsList className="grid h-8 w-full grid-cols-2">
            <TabsTrigger value="edit" className="text-xs">
              {t("edit")}
            </TabsTrigger>
            <TabsTrigger value="history" className="text-xs">
              {t("history")}
            </TabsTrigger>
          </TabsList>
        </div>
        <ScrollArea className="max-h-80 min-h-0 flex-1 lg:max-h-none">
          <TabsContent value="edit" className="m-0 p-3">
            <form
              className="overflow-hidden rounded-xl border bg-background shadow-sm"
              onSubmit={(event) => event.preventDefault()}
            >
              <div className="min-h-56 p-3">
                {component ? (
                  <div className="mb-3 flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/5 px-2.5 py-2 shadow-sm">
                    <UISetTargetThumbnail component={component} />
                    <div className="min-w-0 flex-1">
                      <p className="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
                        {t("target")}
                      </p>
                      <p className="truncate text-sm font-semibold">
                        {component.label}
                      </p>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-xs"
                      aria-label={t("clearSelectedComponent")}
                      title={t("clearSelectedComponent")}
                      onClick={onClearSelection}
                    >
                      <X />
                    </Button>
                  </div>
                ) : null}
                {component ? (
                  <div className="mb-3 grid gap-2">
                    <label className="grid gap-1.5 text-xs font-medium">
                      {t("uiSetComponentLabel")}
                      <Input
                        aria-label={t("uiSetComponentLabel")}
                        value={component.label}
                        onChange={(event) =>
                          onComponentLabelChange(event.target.value)
                        }
                      />
                    </label>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={onRestoreComponent}
                    >
                      <RotateCcw aria-hidden="true" />
                      {t("restoreGeneratedVersion")}
                    </Button>
                  </div>
                ) : (
                  <p className="mb-3 text-xs text-muted-foreground">
                    {t("selectUISetComponent")}
                  </p>
                )}
                <Textarea
                  aria-label={t("editPrompt")}
                  className="min-h-28 resize-none border-0 bg-transparent px-0 py-2 text-sm leading-6 shadow-none focus-visible:border-0 focus-visible:ring-0"
                  placeholder={t("promptPlaceholder")}
                  value={prompt}
                  onChange={(event) => onPromptChange(event.target.value)}
                />
              </div>
              <div className="flex items-center justify-between gap-2 border-t bg-muted/20 px-3 py-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("attachImage")}
                  title={t("attachImage")}
                  disabled
                >
                  <ImagePlus />
                </Button>
                <Button
                  type="submit"
                  size="icon"
                  aria-label={t("sendPrompt")}
                  title={t("sendPrompt")}
                  disabled
                >
                  <ArrowUp />
                </Button>
              </div>
            </form>
          </TabsContent>
          <TabsContent value="history" className="m-0 p-4">
            <InspectorHistory entries={history} />
          </TabsContent>
        </ScrollArea>
      </Tabs>
    </aside>
  );
}

function UISetTargetThumbnail({ component }: { component: UISetComponent }) {
  const Icon = componentIcons[component.kind];
  return (
    <div className="grid size-9 shrink-0 place-items-center overflow-hidden rounded-md border bg-background text-primary shadow-sm">
      <Icon className="size-5" aria-hidden="true" />
    </div>
  );
}
