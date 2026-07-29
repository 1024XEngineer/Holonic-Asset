import {
  ChevronDown,
  Folder,
  ImagePlus,
  Music2,
  Play,
  Plus,
} from "lucide-react";
import { useState } from "react";

import {
  type EditorCharacterAnimation,
  type EditorCharacterAnimationClip,
  type EditorCharacterAnimationGroup,
} from "@/model";
import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";

type AnimationTreeItemsProps = {
  animations: EditorCharacterAnimation[];
  selectedNode: AnimatedSpriteNodeId | null;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onOpenGeneration: () => void;
  onOpenContextMenu: (
    event: React.MouseEvent<HTMLElement>,
    animation: EditorCharacterAnimation,
  ) => void;
  isGenerating: boolean;
};

export function AnimationTreeItems({
  animations,
  selectedNode,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onOpenGeneration,
  onOpenContextMenu,
  isGenerating,
}: AnimationTreeItemsProps) {
  return (
    <>
      <TreeItem
        node="prototype"
        selectedNode={selectedNode}
        onSelect={onSelect}
        icon={<ImagePlus className="size-4" />}
      />
      <AnimationFolder
        count={animations.length}
        onOpenGeneration={onOpenGeneration}
        isGenerating={isGenerating}
      >
        {animations.map((animation) =>
          animation.kind === "group" ? (
            <AnimationGroupTreeItem
              key={animation.id}
              animation={animation}
              selectedNode={selectedNode}
              selectedFrames={selectedFrames}
              onSelect={onSelect}
              onSelectFrame={onSelectFrame}
              onOpenContextMenu={onOpenContextMenu}
            />
          ) : (
            <AnimationTreeItem
              key={animation.id}
              animation={animation}
              selectedNode={selectedNode}
              selectedFrames={selectedFrames}
              onSelect={onSelect}
              onSelectFrame={onSelectFrame}
              onOpenContextMenu={onOpenContextMenu}
            />
          ),
        )}
      </AnimationFolder>
    </>
  );
}

function AnimationFolder({
  count,
  onOpenGeneration,
  isGenerating,
  children,
}: {
  count: number;
  onOpenGeneration: () => void;
  isGenerating: boolean;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="flex w-full items-center gap-2 px-2 py-2 text-xs font-medium text-[#71685d]">
        <Folder className="size-4 text-[#b86b70]" />
        <span className="min-w-0 flex-1 truncate">Animations</span>
        <span className="font-mono text-[10px] text-[#81786d]">{count}</span>
        <button
          type="button"
          aria-label="Generate animation"
          title="Generate animation"
          disabled={isGenerating}
          onClick={onOpenGeneration}
          className="grid size-6 place-items-center rounded-md border border-dashed border-black/15 text-[#7c7368] transition-colors hover:border-[#b86b70]/60 hover:bg-[#b86b70]/5 hover:text-[#b86b70] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#b86b70]"
        >
          <Plus className="size-3.5" />
        </button>
      </div>
      <div className="ml-4 mt-1 space-y-0.5 border-l border-black/10 pl-2">
        {children}
      </div>
    </div>
  );
}

function TreeItem({
  node,
  selectedNode,
  onSelect,
  icon,
}: {
  node: AnimatedSpriteNodeId;
  selectedNode: AnimatedSpriteNodeId | null;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  icon: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(node)}
      className={`group flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left transition-colors ${selectedNode === node ? "bg-black/5 text-[#2d2923]" : "text-[#71685d] hover:bg-black/[.04] hover:text-[#2d2923]"}`}
    >
      <span className="text-[#b86b70]">{icon}</span>
      <span className="min-w-0 flex-1 truncate text-xs font-medium">
        {getAnimatedSpriteNodeLabel(node, [])}
      </span>
    </button>
  );
}

function AnimationTreeItem({
  animation,
  selectedNode,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onOpenContextMenu,
}: {
  animation: EditorCharacterAnimationClip;
  selectedNode: AnimatedSpriteNodeId | null;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onOpenContextMenu?: (
    event: React.MouseEvent<HTMLElement>,
    animation: EditorCharacterAnimationClip,
  ) => void;
}) {
  const node: AnimatedSpriteNodeId = animation.id;
  const [open, setOpen] = useState(false);
  const frames = Array.from(
    { length: animation.frameCount },
    (_, index) => `Frame ${index + 1}`,
  );
  const selected = selectedNode === node;

  return (
    <div onContextMenu={(event) => onOpenContextMenu?.(event, animation)}>
      <div
        className={`flex items-center rounded-lg transition-colors ${selected ? "bg-black/5 text-[#2d2923]" : "text-[#71685d] hover:bg-black/[.04] hover:text-[#2d2923]"}`}
      >
        <button
          type="button"
          onClick={() => onSelect(node)}
          className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left"
        >
          <Play className="size-3.5 text-[#4c7e5e]" />
          <span
            className={`-mx-0.5 inline-flex cursor-pointer rounded p-0.5 transition-all hover:bg-black/[.06] active:scale-90 ${animation.audio ? "text-[#c36d6c] hover:text-[#a84f50]" : "text-[#a9a29a] hover:text-[#71685d]"}`}
          >
            <Music2
              className="size-3.5"
              aria-label={animation.audio ? "Has audio" : "No audio"}
            />
          </span>
          <span className="min-w-0 flex-1 truncate text-xs font-medium">
            {animation.label}
          </span>
        </button>
        <button
          type="button"
          aria-label={`${open ? "Collapse" : "Expand"} ${animation.label}`}
          aria-expanded={open}
          onClick={() => setOpen((current) => !current)}
          className="mr-1 rounded-md p-1.5 text-[#81786d] hover:bg-black/[.05]"
        >
          <ChevronDown
            className={`size-3.5 transition-transform ${open ? "rotate-0" : "-rotate-90"}`}
          />
        </button>
      </div>
      {open ? (
        <div className="ml-4 mt-1 space-y-0.5 border-l border-black/10 pl-2">
          {frames.map((frame, index) => {
            const isSelected = selectedFrames.some(
              (selectedFrame) =>
                selectedFrame.nodeId === node && selectedFrame.index === index,
            );

            return (
              <button
                key={frame}
                type="button"
                aria-pressed={isSelected}
                onClick={() => onSelectFrame(node, index)}
                className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[11px] transition-colors ${isSelected ? "bg-[#b86b70]/10 text-[#8b4e53]" : "text-[#81786d] hover:bg-black/[.04] hover:text-[#51493f]"}`}
              >
                <span className="size-1.5 rounded-full bg-current opacity-70" />
                {frame}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}

function AnimationGroupTreeItem({
  animation,
  selectedNode,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onOpenContextMenu,
}: {
  animation: EditorCharacterAnimationGroup;
  selectedNode: AnimatedSpriteNodeId | null;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onOpenContextMenu: (
    event: React.MouseEvent<HTMLElement>,
    animation: EditorCharacterAnimationGroup,
  ) => void;
}) {
  const [open, setOpen] = useState(true);

  return (
    <div>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        onContextMenu={(event) => onOpenContextMenu(event, animation)}
        className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-[#71685d] transition-colors hover:bg-black/[.04] hover:text-[#2d2923]"
      >
        <Folder className="size-3.5 text-[#b86b70]" />
        <span className="min-w-0 flex-1 truncate text-xs font-medium">
          {animation.label}
        </span>
        <span className="font-mono text-[10px] text-[#81786d]">
          {animation.directions.length}
        </span>
        <ChevronDown
          className={`size-3.5 transition-transform ${open ? "rotate-0" : "-rotate-90"}`}
        />
      </button>
      {open ? (
        <div className="ml-4 mt-1 space-y-0.5 border-l border-black/10 pl-2">
          {animation.directions.map((direction) => (
            <AnimationTreeItem
              key={direction.id}
              animation={direction}
              selectedNode={selectedNode}
              selectedFrames={selectedFrames}
              onSelect={onSelect}
              onSelectFrame={onSelectFrame}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}
