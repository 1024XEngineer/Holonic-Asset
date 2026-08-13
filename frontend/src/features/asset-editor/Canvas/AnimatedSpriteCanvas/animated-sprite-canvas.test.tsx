import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { AnimatedSpriteCanvas } from "./animated-sprite-canvas";

describe("AnimatedSpriteCanvas", () => {
  it("localizes the selected node summary", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <AnimatedSpriteCanvas
          model={{
            prototype: {
              format: "png-sprite-sheet",
              imageUrl: "prototype.png",
              frameWidth: 32,
              frameHeight: 32,
              columns: 4,
              rows: 1,
            },
            animations: [
              { id: "walk", kind: "clip", label: "Walk", frameCount: 4 },
            ],
            selection: { nodeIds: ["walk"], frames: [] },
          }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(html).toContain("Walk selected");
  });
});
