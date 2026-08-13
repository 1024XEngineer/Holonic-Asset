import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import type { SceneryLayer } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { SceneryCanvas } from "./scenery-canvas";

const layers: SceneryLayer[] = [
  {
    id: "sky",
    label: "Sky",
    detail: "Background",
    imageUrl: "/sky.png",
    blendMode: "normal",
  },
  {
    id: "trees",
    label: "Trees",
    detail: "Foreground",
    imageUrl: "/trees.png",
    blendMode: "multiply",
  },
];

describe("SceneryCanvas", () => {
  it("renders selected and hidden layer states", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <SceneryCanvas
          model={{
            layers,
            selectedLayerIds: ["trees"],
            visibleLayerIds: ["sky"],
          }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(html).toContain('aria-pressed="true"');
    expect(html).toContain('aria-pressed="false"');
    expect(html).toContain('aria-hidden="true"');
    expect(html).toContain('tabindex="-1"');
    expect(html).toContain("mix-blend-multiply");
    expect(html).toContain("Trees selected");
  });
});
