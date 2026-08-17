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

const allComponentKinds: UISetComponent[] = [
  {
    id: "input",
    label: "Search",
    kind: "input",
    bounds: { x: 1, y: 1, width: 10, height: 8 },
  },
  {
    id: "badge",
    label: "12",
    kind: "badge",
    bounds: { x: 12, y: 1, width: 10, height: 8 },
  },
  {
    id: "progress",
    label: "Health",
    kind: "progress",
    bounds: { x: 23, y: 1, width: 10, height: 8 },
  },
  {
    id: "toggle",
    label: "Music",
    kind: "toggle",
    bounds: { x: 34, y: 1, width: 10, height: 8 },
  },
  {
    id: "icon",
    label: "Settings",
    kind: "icon",
    bounds: { x: 45, y: 1, width: 10, height: 8 },
  },
  {
    id: "slider",
    label: "Volume",
    kind: "slider",
    bounds: { x: 56, y: 1, width: 20, height: 8 },
  },
];

describe("UISetCanvas", () => {
  it("renders component kinds, clamped bounds, and selection state", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <UISetCanvas
          model={{ components, selectedComponentId: "panel" }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(html).toContain('aria-pressed="true"');
    expect(html).toContain('aria-pressed="false"');
    expect(html).toContain("Panel selected");
    expect(html).toContain("items-start");
    expect(html).toContain("bg-transparent");
    expect(html).toContain("justify-center");
    expect(html).toContain("left:0%");
    expect(html).toContain("width:10%");
    expect(html).toContain("width:0%");
  });

  it("renders every UI Set control kind and the empty state", () => {
    const controls = renderToStaticMarkup(
      withI18n(
        <UISetCanvas
          model={{ components: allComponentKinds, selectedComponentId: null }}
          onEvent={vi.fn()}
        />,
      ),
    );
    const empty = renderToStaticMarkup(
      withI18n(
        <UISetCanvas
          model={{ components: [], selectedComponentId: null }}
          onEvent={vi.fn()}
        />,
      ),
    );

    expect(controls).toContain("Select Search");
    expect(controls).toContain("rounded-full");
    expect(controls).toContain("rotate-45");
    expect(controls).toContain("before:left-2/3");
    expect(empty).toContain("No UI Set components");
  });
});
