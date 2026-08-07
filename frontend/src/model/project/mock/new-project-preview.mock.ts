import type { Perspective } from "../types";

export function createMockProjectPreview({
  description,
  gameType,
  name,
  perspective,
}: {
  description: string;
  gameType: string;
  name: string;
  perspective: Perspective;
}) {
  const canvas = document.createElement("canvas");
  canvas.width = 1280;
  canvas.height = 720;
  const context = canvas.getContext("2d");
  if (!context) return "";

  const isTopDown = perspective === "Top-Down";
  const palette = isTopDown
    ? { sky: "#15253e", ground: "#2e704b", detail: "#f0bb52", ui: "#111827" }
    : { sky: "#30516c", ground: "#537d58", detail: "#e6b968", ui: "#1f2937" };
  const projectName = name.trim() || "Untitled game";
  const summary = description.trim() || `${gameType} project overview`;

  context.fillStyle = palette.sky;
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.fillStyle = "#20354c";
  context.fillRect(0, 300, canvas.width, 180);
  context.fillStyle = palette.ground;
  context.fillRect(0, 480, canvas.width, 240);

  for (let x = 0; x < canvas.width; x += 64) {
    context.fillStyle = x % 128 === 0 ? "#3b8658" : "#347a51";
    context.fillRect(x, 560, 62, 58);
  }

  context.fillStyle = palette.detail;
  context.fillRect(570, 360, 140, 180);
  context.fillStyle = "#f5d78f";
  context.fillRect(600, 320, 80, 65);
  context.fillStyle = "#182536";
  context.fillRect(615, 410, 50, 80);

  context.fillStyle = palette.ui;
  context.fillRect(48, 48, 310, 126);
  context.fillStyle = "#ffffff";
  context.font = "600 34px system-ui";
  context.fillText(projectName, 76, 98);
  context.fillStyle = "#cbd5e1";
  context.font = "24px system-ui";
  context.fillText(perspective || "Perspective", 76, 138);

  context.fillStyle = "#ffffff";
  context.font = "28px system-ui";
  context.fillText(summary.slice(0, 76), 48, 670);

  return canvas.toDataURL("image/png");
}
