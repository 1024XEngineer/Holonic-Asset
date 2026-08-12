import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import type { UISetComponent } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { UISetCanvas } from "./ui-set-canvas";

const components: UISetComponent[] = [
  {
    id: "panel",
    label: "Panel",
    kind: "panel",
    bounds: {
      x: Number.NaN,
      y: -10,
      width: Number.POSITIVE_INFINITY,
      height: 150,
    },
  },
  {
    id: "label",
    label: "Label",
    kind: "label",
    bounds: { x: 90, y: 95, width: 40, height: 20 },
  },
  {
    id: "button",
    label: "Button",
    kind: "button",
    bounds: { x: 120, y: 120, width: 50, height: 50 },
  },
];

describe("UISetCanvas", () => {
  it("renders component kinds, clamped bounds, and selection state", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <UISetCanvas
          model={{ components, selectedComponentIds: ["panel", "button"] }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(html).toContain('aria-pressed="true"');
    expect(html).toContain('aria-pressed="false"');
    expect(html).toContain("Panel, Button selected");
    expect(html).toContain("items-start");
    expect(html).toContain("bg-transparent");
    expect(html).toContain("justify-center");
    expect(html).toContain("left:0%");
    expect(html).toContain("width:10%");
    expect(html).toContain("width:0%");
  });
});
