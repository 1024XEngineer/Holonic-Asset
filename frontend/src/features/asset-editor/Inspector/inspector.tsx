import { History, Sparkles } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import type { AssetRevision, CharacterAnimation } from "@/model";

import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";

export function Inspector({
  selectedNodes,
  selectedFrames,
  prompt,
  onPromptChange,
  history,
  animations,
}: {
  selectedNodes: AnimatedSpriteNodeId[];
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  prompt: string;
  onPromptChange: (value: string) => void;
  history: AssetRevision[];
  animations: CharacterAnimation[];
}) {
  const [activeTab, setActiveTab] = useState<"brief" | "history">("brief");

  return (
    <aside className="flex w-full shrink-0 flex-col border-t bg-background lg:w-72 lg:border-l lg:border-t-0">
      <div
        className="grid grid-cols-2 border-b"
        role="tablist"
        aria-label="Inspector"
      >
        <InspectorTab
          active={activeTab === "brief"}
          id="brief"
          label="Creative brief"
          onClick={() => setActiveTab("brief")}
        />
        <InspectorTab
          active={activeTab === "history"}
          id="history"
          label="History"
          onClick={() => setActiveTab("history")}
        />
      </div>
      <ScrollArea className="max-h-72 flex-1 lg:max-h-none">
        <div className="p-4">
          {activeTab === "brief" ? (
            <div role="tabpanel" aria-labelledby="brief-tab" id="brief-panel">
              <SelectionSummary
                selectedNodes={selectedNodes}
                selectedFrames={selectedFrames}
                animations={animations}
              />
              <label className="grid gap-2 text-xs font-medium text-muted-foreground">
                Description
                <Textarea
                  className="min-h-44 resize-none text-sm leading-6"
                  placeholder="Describe the asset's appearance and motion..."
                  value={prompt}
                  onChange={(event) => onPromptChange(event.target.value)}
                />
              </label>
              <div className="mt-3 flex items-center gap-2 text-[11px] text-muted-foreground">
                <Sparkles className="size-3.5 text-primary" />
                Draft context
              </div>
            </div>
          ) : (
            <SaveHistory entries={history} />
          )}
        </div>
      </ScrollArea>
    </aside>
  );
}

function InspectorTab({
  active,
  id,
  label,
  onClick,
}: {
  active: boolean;
  id: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      aria-controls={`${id}-panel`}
      aria-selected={active}
      id={`${id}-tab`}
      role="tab"
      type="button"
      onClick={onClick}
      className={`border-b-2 px-2 py-3 text-center text-xs font-semibold transition-colors ${active ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"}`}
    >
      {label}
    </button>
  );
}

function SelectionSummary({
  selectedNodes,
  selectedFrames,
  animations,
}: {
  selectedNodes: AnimatedSpriteNodeId[];
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  animations: CharacterAnimation[];
}) {
  let value = "Entire asset";
  if (selectedFrames.length > 0) {
    const nodeId = selectedFrames[0]?.nodeId;
    const frames = selectedFrames.map((frame) => frame.index + 1).join(", ");
    value = nodeId
      ? `${getAnimatedSpriteNodeLabel(nodeId, animations)} / ${frames}`
      : frames;
  } else if (selectedNodes.length > 0) {
    value = selectedNodes
      .map((nodeId) => getAnimatedSpriteNodeLabel(nodeId, animations))
      .join(", ");
  }

  return (
    <div className="mb-4 rounded-lg border bg-muted/40 p-3 text-xs">
      <span className="text-muted-foreground">Target</span>
      <p className="mt-1 truncate font-medium">{value}</p>
    </div>
  );
}

function SaveHistory({ entries }: { entries: AssetRevision[] }) {
  if (entries.length === 0) {
    return (
      <div className="py-8 text-center text-xs text-muted-foreground">
        <History className="mx-auto mb-2 size-6" />
        No saved records
      </div>
    );
  }

  return (
    <div
      className="space-y-3"
      role="tabpanel"
      aria-labelledby="history-tab"
      id="history-panel"
    >
      {entries.map((entry) => (
        <div
          key={entry.id}
          className="rounded-lg border bg-muted/30 p-3 text-xs"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="font-mono text-[11px] text-muted-foreground">
              {entry.version}
            </span>
            {entry.isCurrent ? (
              <Badge variant="secondary">Current</Badge>
            ) : null}
          </div>
          <p className="mt-2 font-medium leading-5">{entry.description}</p>
        </div>
      ))}
    </div>
  );
}
