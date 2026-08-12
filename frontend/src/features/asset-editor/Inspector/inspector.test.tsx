import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import type { CharacterAnimation } from "@/model";
import { withI18n } from "@/testing/with-i18n";

import { Inspector } from "./inspector";
import { getInspectorTargetSummary } from "./inspector-target";

const animations: CharacterAnimation[] = [
  {
    kind: "clip",
    id: "idle-front",
    label: "Idle Front",
    frameCount: 4,
  },
];

describe("Inspector", () => {
  it("renders the AI composer controls without the old draft label", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <Inspector
          selectedNodes={[]}
          selectedFrames={[]}
          prompt="Refine the silhouette"
          onPromptChange={() => undefined}
          history={[]}
          animations={animations}
          onSubmit={() => undefined}
          onClearSelection={() => undefined}
        />,
      ),
    );

    expect(html).toContain("Entire asset");
    expect(html).toContain("Edit");
    expect(html).toContain("What would you like to change?");
    expect(html).toContain("Attach image");
    expect(html).toContain("Send prompt");
    expect(html).not.toContain("Draft context");
  });

  it("describes selected animation frames inside the composer", () => {
    expect(
      getInspectorTargetSummary(
        ["idle-front"],
        [
          { nodeId: "idle-front", index: 0 },
          { nodeId: "idle-front", index: 2 },
        ],
        animations,
      ),
    ).toEqual({ label: "Idle Front", detail: "Frames 1, 3" });
  });
});
