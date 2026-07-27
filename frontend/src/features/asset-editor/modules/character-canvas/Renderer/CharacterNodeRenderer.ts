import { Container, Graphics, Text } from "pixi.js";

import type { EditorCharacterAnimation } from "@/features/assets/domain";
import {
  findCharacterAnimation,
  getCharacterNodeLabel,
  type NodeId,
} from "../character-node";
import {
  ANIMATION_NODES,
  type CanvasPosition,
} from "../CharacterCanvas.constants";
import { getCharacterNodeLayout } from "../Interaction/CharacterStageGeometry";
import { STAGE_ACCENT } from "../Runtime/CharacterStage.constants";
import type { Bounds } from "../Runtime/CharacterCanvas.types";

export function drawCharacterNode({
  node,
  position,
  selected,
  selectedFrames,
  expanded,
  playing,
  previewFrame,
  animations,
}: {
  node: NodeId;
  position: CanvasPosition;
  selected: boolean;
  selectedFrames: number[];
  expanded: boolean;
  playing: boolean;
  previewFrame: number;
  animations: EditorCharacterAnimation[];
}) {
  const container = new Container({ x: position.x, y: position.y });
  const layout = getCharacterNodeLayout(
    node,
    { x: 0, y: 0 },
    expanded,
    animations,
  );
  drawLabel(
    container,
    getCharacterNodeLabel(node, animations),
    layout.bounds.width,
    selected,
  );

  if (expanded) {
    layout.frames.forEach((frame, index) => {
      drawFrame(container, frame, index, selectedFrames.includes(index));
    });
  } else {
    drawCharacter(container, 64, 48, previewFrame, 8);
  }

  const audio = findCharacterAnimation(node, animations)?.audio;
  if (audio && !expanded) drawAudioWaveform(container, audio.label);

  if (ANIMATION_NODES.has(node)) {
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
  drawCharacter(container, bounds.x + 28, bounds.y + 8, index, 4);
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
