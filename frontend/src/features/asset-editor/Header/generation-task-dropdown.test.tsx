// @vitest-environment happy-dom

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { GenerationTaskDropdown } from "./generation-task-dropdown";

afterEach(cleanup);

describe("GenerationTaskDropdown", () => {
  it("renders nothing without generation tasks", () => {
    const { container } = render(
      withI18n(<GenerationTaskDropdown tasks={[]} />),
    );

    expect(container.innerHTML).toBe("");
  });

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

  it("shows queued, processing, and failed task details", async () => {
    const user = userEvent.setup();
    render(
      withI18n(
        <GenerationTaskDropdown
          tasks={[
            {
              id: "run-1",
              name: "Walk",
              prompt: "A relaxed walk",
              status: "queued",
            },
            {
              id: "run-2",
              name: "Run",
              prompt: "A fast run",
              status: "processing",
            },
            {
              id: "run-3",
              name: "Jump",
              prompt: "A high jump",
              status: "failed",
              error: "Video provider rejected the request",
            },
          ]}
        />,
      ),
    );

    const trigger = screen.getByRole("button", {
      name: /3 generations active/,
    });
    expect(trigger.innerHTML).toContain("animate-spin");
    await user.click(trigger);

    expect(await screen.findByText("Queued")).toBeTruthy();
    expect(screen.getByText("Generating")).toBeTruthy();
    expect(screen.getByText("Failed")).toBeTruthy();
    expect(
      screen.getByText("Video provider rejected the request"),
    ).toBeTruthy();
  });
});
