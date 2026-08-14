import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { GenerationTaskDropdown } from "./generation-task-dropdown";

describe("GenerationTaskDropdown", () => {
  it("shows a failed task without an active loading spinner", () => {
    const html = renderToStaticMarkup(
      withI18n(
        <GenerationTaskDropdown
          tasks={[
            {
              id: "run-1",
              name: "Walk",
              prompt: "A relaxed walk",
              status: "failed",
              error: "Video provider rejected the request",
            },
          ]}
        />,
      ),
    );

    expect(html).toContain("1 generation failed");
    expect(html).not.toContain("animate-spin");
  });
});
