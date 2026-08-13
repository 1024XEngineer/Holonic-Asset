import { Container, Graphics, Text } from "pixi.js";
import type { CharacterAnimation, CharacterSpriteSheet } from "@/model";
import { getAnimatedSpritePixelScale } from "../animated-sprite-scale";
import { getGridRowCount } from "../grid-row-count";
import { getSpriteSheetFrameCount } from "../sprite-sheet-grid";
import {
  getAnimatedSpriteAnimation,
  getAnimatedSpriteNodeLabel,
  type NodeId,
} from "../animated-sprite-node";
import type { CanvasPosition } from "../AnimatedSpriteCanvas.constants";
import { getAnimatedSpriteNodeLayout } from "../Interaction/AnimatedSpriteStageGeometry";
import {
  FRAME_SIZE,
  STAGE_ACCENT,
} from "../Runtime/AnimatedSpriteStage.constants";
import type { Bounds } from "../Runtime/AnimatedSpriteCanvas.types";
import { drawSpriteSheetFrame } from "./SpriteSheetFrameRenderer";
import type { SpriteSheetFrameTextureCache } from "./SpriteSheetFrameTextureCache";

const COLLAPSED_PREVIEW_Y = 80;

export function drawAnimatedSpriteNode({
  node,
  frameTextures,
  position,
  selected,
  selectedFrames,
  expanded,
  playing,
  previewFrame,
  animations,
  prototype,
  unavailableTextureUrls,
}: {
  node: NodeId;
  frameTextures: SpriteSheetFrameTextureCache;
  position: CanvasPosition;
  selected: boolean;
  selectedFrames: number[];
  expanded: boolean;
  playing: boolean;
  previewFrame: number;
  animations: CharacterAnimation[];
  prototype: CharacterSpriteSheet;
  unavailableTextureUrls?: ReadonlySet<string>;
}) {
  const container = new Container({ x: position.x, y: position.y });
  const layout = getAnimatedSpriteNodeLayout(
    node,
    { x: 0, y: 0 },
    expanded,
    prototype,
    animations,
  );
  const animation = getAnimatedSpriteAnimation(node, animations);
  drawLabel(
    container,
    getAnimatedSpriteNodeLabel(node, animations),
    layout.bounds.width,
    selected,
  );

  const spriteSheet = node === "prototype" ? prototype : animation?.spriteSheet;
  const pixelScale = getAnimatedSpritePixelScale(
    spriteSheet ?? prototype,
    FRAME_SIZE,
  );
  if (expanded) {
    layout.frames.forEach((frame, index) =>
      drawFrame(
        container,
        frame,
        index,
        selectedFrames.includes(index),
        spriteSheet,
        pixelScale,
        frameTextures,
        unavailableTextureUrls,
      ),
    );
  } else if (
    node === "prototype" &&
    hasAvailablePrototypeFrame(prototype, unavailableTextureUrls)
  ) {
    drawSpriteSheetPreview(
      container,
      frameTextures,
      prototype,
      layout.bounds.width,
      pixelScale,
    );
  } else if (
    spriteSheet?.imageUrl &&
    !unavailableTextureUrls?.has(spriteSheet.imageUrl)
  ) {
    drawSpriteSheetFrame({
      container,
      frameTextures,
      spriteSheet,
      frame: previewFrame,
      bounds: {
        x: (layout.bounds.width - FRAME_SIZE) / 2,
        y: COLLAPSED_PREVIEW_Y,
        width: FRAME_SIZE,
        height: FRAME_SIZE,
      },
      pixelScale,
    });
  }

  if (animation?.audio && !expanded) {
    drawAudioWaveform(container, animation.audio.label);
  }

  if (animation) {
    const play = layout.playControl!;
    const expand = layout.expandControl!;
    drawControl(
      container,
      play.x,
      play.y,
      play.width,
      play.height,
      playing ? "Pause" : "Play",
      playing ? "||" : ">",
      !layout.playEnabled,
    );
    drawControl(
      container,
      expand.x,
      expand.y,
      expand.width,
      expand.height,
      expanded ? "Collapse" : "Expand",
      expanded ? "-" : "+",
      false,
    );
  }
  return container;
}

function hasAvailablePrototypeFrame(
  prototype: CharacterSpriteSheet,
  unavailableTextureUrls?: ReadonlySet<string>,
) {
  const urls = prototype.frameUrls?.length
    ? prototype.frameUrls
    : [prototype.imageUrl];
  return urls.some((url) => url && !unavailableTextureUrls?.has(url));
}

function drawLabel(
  container: Container,
  value: string,
  width: number,
  selected: boolean,
) {
  const label = new Text({
    text: value,
    style: {
      fill: 0x51493f,
      fontFamily: "ui-monospace, monospace",
      fontSize: 11,
      fontWeight: "500",
    },
  });
  container.addChild(
    new Graphics()
      .roundRect(width / 2 - label.width / 2 - 10, 7, label.width + 20, 24, 5)
      .fill({ color: 0xffffff, alpha: 0.92 })
      .stroke({
        color: selected ? STAGE_ACCENT : 0x000000,
        alpha: selected ? 0.9 : 0.1,
        width: 1,
      }),
  );
  label.position.set(width / 2 - label.width / 2, 12);
  container.addChild(label);
}

function drawFrame(
  container: Container,
  bounds: Bounds,
  index: number,
  selected: boolean,
  spriteSheet: CharacterSpriteSheet | undefined,
  pixelScale: number,
  frameTextures: SpriteSheetFrameTextureCache,
  unavailableTextureUrls?: ReadonlySet<string>,
) {
  if (selected)
    container.addChild(
      new Graphics()
        .roundRect(
          bounds.x - 3,
          bounds.y - 3,
          bounds.width + 6,
          bounds.height + 6,
          5,
        )
        .stroke({ color: STAGE_ACCENT, width: 2 }),
    );
  const imageUrl = spriteSheet?.frameUrls?.[index] ?? spriteSheet?.imageUrl;
  if (spriteSheet && imageUrl && !unavailableTextureUrls?.has(imageUrl)) {
    drawSpriteSheetFrame({
      container,
      frameTextures,
      spriteSheet,
      frame: index,
      bounds,
      pixelScale,
    });
  }
}

function drawSpriteSheetPreview(
  container: Container,
  frameTextures: SpriteSheetFrameTextureCache,
  spriteSheet: CharacterSpriteSheet,
  containerWidth: number,
  pixelScale: number,
) {
  const frameCount = getSpriteSheetFrameCount(spriteSheet);
  const columns = frameCount === 1 ? 1 : 2;
  const rows = getGridRowCount(frameCount, columns);
  const gap = 8;
  const width = columns * FRAME_SIZE + (columns - 1) * gap;
  const height = rows * FRAME_SIZE + (rows - 1) * gap;
  const startX = (containerWidth - width) / 2;
  const startY =
    frameCount === 1 ? COLLAPSED_PREVIEW_Y : 48 + (200 - height) / 2;
  for (let frame = 0; frame < frameCount; frame += 1)
    drawSpriteSheetFrame({
      container,
      frameTextures,
      spriteSheet,
      frame,
      bounds: {
        x: startX + (frame % columns) * (FRAME_SIZE + gap),
        y: startY + Math.floor(frame / columns) * (FRAME_SIZE + gap),
        width: FRAME_SIZE,
        height: FRAME_SIZE,
      },
      pixelScale,
    });
}

function drawControl(
  container: Container,
  x: number,
  y: number,
  width: number,
  height: number,
  label: string,
  icon: string,
  disabled: boolean,
) {
  container.addChild(
    new Graphics()
      .roundRect(x, y, width, height, 5)
      .fill({ color: 0xffffff, alpha: 0.4 })
      .stroke({ color: 0x000000, alpha: 0.1, width: 1 }),
  );
  const text = new Text({
    text: `${icon}  ${label}`,
    style: {
      fill: 0x51493f,
      fontFamily: "ui-sans-serif, sans-serif",
      fontSize: 11,
      fontWeight: "500",
    },
    alpha: disabled ? 0.35 : 1,
  });
  text.position.set(x + width / 2 - text.width / 2, y + 9);
  container.addChild(text);
}

function drawAudioWaveform(container: Container, label: string) {
  container.addChild(
    new Graphics({ label: `Attached audio: ${label}` })
      .roundRect(20, 210, 184, 34, 6)
      .fill({ color: 0xffffff, alpha: 0.7 })
      .stroke({ color: 0x000000, alpha: 0.08, width: 1 }),
  );
  const waveform = new Graphics({ label: `Attached audio: ${label}` });
  const bars = [
    10, 18, 26, 14, 30, 20, 12, 24, 16, 28, 14, 22, 10, 20, 26, 16, 24, 12, 22,
    28, 18, 10, 24, 14,
  ];
  bars.forEach((height, index) => {
    waveform
      .roundRect(30 + index * 7, 210 + (34 - height) / 2, 2, height, 1)
      .fill({ color: 0x81786d, alpha: 0.45 });
  });
  container.addChild(waveform);
}
