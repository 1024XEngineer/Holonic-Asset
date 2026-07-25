import { History, Sparkles } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { AssetRecord } from "@/domain/asset";
import {
  characterNodeMeta,
  type CharacterCanvasNodeId,
} from "@/modules/character-canvas";

type InspectorProps = {
  selectedNodes: CharacterCanvasNodeId[];
  selectedFrames: Array<{ nodeId: CharacterCanvasNodeId; index: number }>;
  prompt: string;
  onPromptChange: (value: string) => void;
  onAction: (message: string) => void;
  history: AssetRecord[];
  selectedItems?: string[];
};

export function Inspector({
  selectedNodes,
  selectedFrames,
  prompt,
  onPromptChange,
  onAction,
  history,
  selectedItems,
}: InspectorProps) {
  const [activeTab, setActiveTab] = useState<"ai-edit" | "history">("ai-edit");

  return (
    <aside className="flex w-full shrink-0 flex-col border-t border-black/10 bg-[#ffffff] lg:w-[18rem] lg:border-l lg:border-t-0">
      <ScrollArea className="max-h-[24rem] flex-1 lg:max-h-none">
        <div
          aria-label="Inspector sections"
          className="flex border-b border-black/10"
          role="tablist"
        >
          <InspectorTab
            active={activeTab === "ai-edit"}
            id="ai-edit"
            label="AI Edit"
            onClick={() => setActiveTab("ai-edit")}
          />
          <InspectorTab
            active={activeTab === "history"}
            id="history"
            label="History"
            onClick={() => setActiveTab("history")}
          />
        </div>
        <div className="p-4">
          {activeTab === "ai-edit" ? (
            <div
              role="tabpanel"
              aria-labelledby="ai-edit-tab"
              id="ai-edit-panel"
            >
              <SelectionSummary
                selectedNodes={selectedNodes}
                selectedFrames={selectedFrames}
                selectedItems={selectedItems}
              />
              <label className="grid gap-2 text-sm font-medium text-[#51493f]">
                Description
                <div className="overflow-hidden rounded-2xl border border-black/10 bg-[#ffffff] shadow-sm focus-within:ring-3 focus-within:ring-[#b86b70]/25">
                  <textarea
                    autoFocus
                    className="min-h-44 w-full resize-none bg-transparent px-4 py-3 text-sm leading-6 text-[#403a32] outline-none placeholder:text-[#81786d]"
                    placeholder="Describe how you want to update this asset..."
                    value={prompt}
                    onChange={(event) => onPromptChange(event.target.value)}
                  />
                  <div className="flex justify-end p-3 pt-0">
                    <Button
                      disabled={!prompt.trim()}
                      onClick={() => onAction("Generation queued")}
                    >
                      <Sparkles />
                      Generate asset
                    </Button>
                  </div>
                </div>
              </label>
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
      className={`flex-1 border-b-2 py-3 text-center text-xs font-semibold uppercase tracking-[0.14em] transition-colors ${active ? "border-[#b86b70] text-[#b86b70]" : "border-transparent text-[#81786d] hover:text-[#2d2923]"}`}
    >
      {label}
    </button>
  );
}

function SelectionSummary({
  selectedNodes,
  selectedFrames,
  selectedItems,
}: {
  selectedNodes: CharacterCanvasNodeId[];
  selectedFrames: Array<{ nodeId: CharacterCanvasNodeId; index: number }>;
  selectedItems?: string[];
}) {
  if (selectedItems) {
    return (
      <div className="mb-4 rounded-xl border border-black/10 bg-[#f7f5f0] p-3 text-xs text-[#51493f]">
        <span className="font-semibold text-[#2d2923]">Target:</span>{" "}
        {selectedItems.length ? selectedItems.join(", ") : "Entire asset"}
      </div>
    );
  }

  if (selectedFrames.length > 0) {
    const node = selectedFrames[0]?.nodeId;

    return (
      <div className="mb-4 rounded-xl border border-black/10 bg-[#f7f5f0] p-3 text-xs text-[#51493f]">
        <span className="font-semibold text-[#2d2923]">Target:</span>{" "}
        {node ? `${characterNodeMeta[node].label} (` : ""}
        {selectedFrames
          .map((selectedFrame) => `Frame ${selectedFrame.index + 1}`)
          .join(", ")}
        {node ? ")" : ""}
      </div>
    );
  }

  if (selectedNodes.length > 0) {
    return (
      <div className="mb-4 rounded-xl border border-black/10 bg-[#f7f5f0] p-3 text-xs text-[#51493f]">
        <span className="font-semibold text-[#2d2923]">Target:</span>{" "}
        {selectedNodes.map((node) => characterNodeMeta[node].label).join(", ")}
      </div>
    );
  }

  return (
    <div className="mb-4 rounded-xl border border-black/10 bg-[#f7f5f0] p-3 text-xs text-[#81786d]">
      Select nodes or frames on canvas to target your edit.
    </div>
  );
}

function SaveHistory({ entries }: { entries: AssetRecord[] }) {
  if (entries.length === 0) {
    return (
      <div className="py-8 text-center text-xs text-[#81786d]">
        <History className="mx-auto mb-2 size-6 text-[#b86b70]" />
        No save history yet.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {entries.map((entry) => (
        <div
          key={entry.id}
          className="rounded-xl border border-black/10 bg-[#f7f5f0] p-3 text-xs text-[#51493f]"
        >
          <div className="flex items-center justify-between text-[11px] text-[#81786d]">
            <span>{entry.version}</span>
          </div>
          <p className="mt-1 font-medium text-[#2d2923]">{entry.description}</p>
          <p className="mt-1 text-[11px] text-[#786f64]">
            {entry.isCurrent ? "Current record" : "Saved record"}
          </p>
        </div>
      ))}
    </div>
  );
}
