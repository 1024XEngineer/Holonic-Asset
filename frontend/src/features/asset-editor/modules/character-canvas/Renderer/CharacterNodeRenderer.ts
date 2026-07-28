import { Container, Graphics, Rectangle, Sprite, Text, Texture } from "pixi.js";

import type {
  EditorCharacterAnimation,
  EditorCharacterSpriteSheet,
} from "../../../domain";
import {
  getCharacterCanvasAnimation,
  getCharacterNodeLabel,
  type CharacterDirectionMap,
  type NodeId,
} from "../character-node";
import type { CanvasPosition } from "../CharacterCanvas.constants";
import { getCharacterNodeLayout } from "../Interaction/CharacterStageGeometry";
import {
  FRAME_SIZE,
  getCharacterPixelScale,
  STAGE_ACCENT,
} from "../Runtime/CharacterStage.constants";
import type { Bounds } from "../Runtime/CharacterCanvas.types";

const COLLAPSED_PREVIEW_Y = 80;
const PROTOTYPE_FRAME_GAP = 8;

export function drawCharacterNode({
  node,
  position,
  selected,
  selectedFrames,
  expanded,
  playing,
  previewFrame,
  activeDirections,
  animations,
  prototype,
  unavailableTextureUrls,
}: {
  node: NodeId;
  position: CanvasPosition;
  selected: boolean;
  selectedFrames: number[];
  expanded: boolean;
  playing: boolean;
  previewFrame: number;
  activeDirections: CharacterDirectionMap;
  animations: EditorCharacterAnimation[];
  prototype: EditorCharacterSpriteSheet;
  unavailableTextureUrls?: ReadonlySet<string>;
}) {
  const container = new Container({ x: position.x, y: position.y });
  const layout = getCharacterNodeLayout(
    node,
    { x: 0, y: 0 },
    expanded,
    animations,
    activeDirections,
  );
  drawLabel(
    container,
    getCharacterNodeLabel(node, animations, activeDirections),
    layout.bounds.width,
    selected,
  );

  const animation = getCharacterCanvasAnimation(
    node,
    animations,
    activeDirections,
  );
  const nodeSpriteSheet =
    node === "prototype" ? prototype : animation?.spriteSheet;
  const pixelScale = getCharacterPixelScale(nodeSpriteSheet ?? prototype);
  if (expanded) {
    layout.frames.forEach((frame, index) => {
      drawFrame(
        container,
        frame,
        index,
        selectedFrames.includes(index),
        animation?.spriteSheet,
        pixelScale,
        unavailableTextureUrls,
      );
    });
  } else if (
    node === "prototype" &&
    prototype.imageUrl &&
    !unavailableTextureUrls?.has(prototype.imageUrl)
  ) {
    drawSpriteSheetPreview(
      container,
      prototype,
      layout.bounds.width,
      pixelScale,
    );
  } else if (
    animation?.spriteSheet?.imageUrl &&
    !unavailableTextureUrls?.has(animation.spriteSheet.imageUrl)
  ) {
    drawSpriteSheetFrame(
      container,
      animation.spriteSheet,
      previewFrame,
      (layout.bounds.width - FRAME_SIZE) / 2,
      COLLAPSED_PREVIEW_Y,
      FRAME_SIZE,
      FRAME_SIZE,
      pixelScale,
    );
  } else {
    drawCharacterPlaceholder(
      container,
      {
        x: (layout.bounds.width - FRAME_SIZE) / 2,
        y: COLLAPSED_PREVIEW_Y,
        width: FRAME_SIZE,
        height: FRAME_SIZE,
      },
      previewFrame,
    );
  }

  const audio = animation?.audio;
  if (audio && !expanded) drawAudioWaveform(container, audio.label);

  if (animation) {
    const playControl = layout.playControl!;
    const expandControl = layout.expandControl!;
    drawControl(
      container,
      playControl.x,
      playControl.y,
      playControl.width,
      playControl.height,
      playing ? "Pause" : "Play",
      playing ? "||" : ">",
      !layout.playEnabled,
    );
    drawControl(
      container,
      expandControl.x,
      expandControl.y,
      expandControl.width,
      expandControl.height,
      expanded ? "Collapse" : "Expand",
      expanded ? "-" : "+",
      false,
    );
    const group = animations.find((candidate) => candidate.id === node);
    if (group && "directions" in group && group.directions.length > 1) {
      const switchControl = layout.switchControl!;
      drawControl(
        container,
        switchControl.x,
        switchControl.y,
        switchControl.width,
        switchControl.height,
        "Switch",
        "<>",
        false,
      );
    }
  }

  return container;
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
  spriteSheet?: EditorCharacterSpriteSheet,
  pixelScale = 1,
  unavailableTextureUrls?: ReadonlySet<string>,
) {
  if (selected) {
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
  }

  if (
    spriteSheet?.imageUrl &&
    !unavailableTextureUrls?.has(spriteSheet.imageUrl)
  ) {
    drawSpriteSheetFrame(
      container,
      spriteSheet,
      index,
      bounds.x,
      bounds.y,
      bounds.width,
      bounds.height,
      pixelScale,
    );
  } else {
    drawCharacterPlaceholder(container, bounds, index);
  }
}

function drawCharacterPlaceholder(
  container: Container,
  bounds: Bounds,
  frame: number,
) {
  const size = Math.floor(Math.min(bounds.width / 12, bounds.height / 17));
  drawCharacter(
    container,
    bounds.x + (bounds.width - 12 * size) / 2,
    bounds.y + (bounds.height - 17 * size) / 2,
    frame,
    size,
  );
}

function drawCharacter(
  container: Container,
  offsetX: number,
  offsetY: number,
  frame: number,
  size: number,
) {
  const pixels = new Graphics();
  for (let index = 0; index < 12 * 17; index += 1) {
    const x = index % 12;
    const y = Math.floor(index / 12);
    const isHead = y >= 2 && y <= 5 && x >= 3 && x <= 8;
    const isHair = y >= 1 && y <= 3 && x >= 2 && x <= 9;
    const isBody = y >= 6 && y <= 11 && x >= 3 && x <= 8;
    const isScarf = y === 7 && x >= 2 && x <= 9;
    const isLeg =
      y >= 12 && y <= 15 && ((x >= 3 && x <= 4) || (x >= 7 && x <= 8));
    const isShadow = y === 16 && x >= 2 && x <= 9;
    let color: number | null = null;
    let alpha = 1;
    if (isHair) color = 0x5a3d32;
    if (isHead) color = 0xe8aa7d;
    if (isBody) color = 0x5e7892;
    if (isScarf) color = 0xd58a57;
    if (isLeg) color = 0x3d4a62;
    if (isShadow) {
      color = 0x735d4a;
      alpha = 0.34;
    }
    if (frame % 2 === 1 && y === 12 && x === 4) color = 0xf09b5b;
    if (frame % 3 === 2 && y === 13 && x === 8) color = 0x91c7a5;
    if (color !== null) {
      pixels
        .roundRect(
          offsetX + x * size,
          offsetY + y * size,
          size - 1,
          size - 1,
          1,
        )
        .fill({ color, alpha });
    }
  }
  container.addChild(pixels);
}

const spriteSheetFrameTextures = new Map<string, Texture>();

function drawSpriteSheetPreview(
  container: Container,
  spriteSheet: EditorCharacterSpriteSheet,
  containerWidth: number,
  pixelScale: number,
) {
  const frameCount = spriteSheet.columns * spriteSheet.rows;
  const previewColumns = frameCount === 1 ? 1 : 2;
  const previewRows = Math.ceil(frameCount / previewColumns);
  const previewWidth =
    previewColumns * FRAME_SIZE + (previewColumns - 1) * PROTOTYPE_FRAME_GAP;
  const previewHeight =
    previewRows * FRAME_SIZE + (previewRows - 1) * PROTOTYPE_FRAME_GAP;
  const startX = (containerWidth - previewWidth) / 2;
  const startY =
    frameCount === 1 ? COLLAPSED_PREVIEW_Y : 48 + (200 - previewHeight) / 2;

  for (let frame = 0; frame < frameCount; frame += 1) {
    drawSpriteSheetFrame(
      container,
      spriteSheet,
      frame,
      startX + (frame % previewColumns) * (FRAME_SIZE + PROTOTYPE_FRAME_GAP),
      startY +
        Math.floor(frame / previewColumns) * (FRAME_SIZE + PROTOTYPE_FRAME_GAP),
      FRAME_SIZE,
      FRAME_SIZE,
      pixelScale,
    );
  }
}

function drawSpriteSheetFrame(
  container: Container,
  spriteSheet: EditorCharacterSpriteSheet,
  frame: number,
  x: number,
  y: number,
  width: number,
  height: number,
  pixelScale: number,
) {
  const column = frame % spriteSheet.columns;
  const row = spriteSheet.row ?? Math.floor(frame / spriteSheet.columns);
  const cacheKey = [
    spriteSheet.imageUrl,
    column,
    row,
    spriteSheet.frameWidth,
    spriteSheet.frameHeight,
  ].join(":");
  let texture = spriteSheetFrameTextures.get(cacheKey);

  if (!texture) {
    const source = Texture.from(spriteSheet.imageUrl).source;
    texture = new Texture({
      source,
      frame: new Rectangle(
        column * spriteSheet.frameWidth,
        row * spriteSheet.frameHeight,
        spriteSheet.frameWidth,
        spriteSheet.frameHeight,
      ),
    });
    spriteSheetFrameTextures.set(cacheKey, texture);
  }

  drawSprite(
    container,
    texture,
    spriteSheet.frameWidth,
    spriteSheet.frameHeight,
    x,
    y,
    width,
    height,
    pixelScale,
  );
}

function drawSprite(
  container: Container,
  texture: Texture,
  sourceWidth: number,
  sourceHeight: number,
  x: number,
  y: number,
  width: number,
  height: number,
  pixelScale: number,
) {
  const sprite = new Sprite(texture);
  const renderedWidth = sourceWidth * pixelScale;
  const renderedHeight = sourceHeight * pixelScale;
  const centeredX = x + (width - renderedWidth) / 2;
  const centeredY = y + (height - renderedHeight) / 2;
  sprite.position.set(
    snapToPixelGrid(container.x + centeredX, pixelScale) - container.x,
    snapToPixelGrid(container.y + centeredY, pixelScale) - container.y,
  );
  sprite.scale.set(pixelScale);
  container.addChild(sprite);
}

function snapToPixelGrid(value: number, pixelScale: number) {
  return Math.round(value / pixelScale) * pixelScale;
}

function drawAudioWaveform(container: Container, label: string) {
  container.addChild(
    new Graphics()
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
