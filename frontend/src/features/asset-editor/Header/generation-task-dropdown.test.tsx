// @vitest-environment happy-dom

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";

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

  it("offers apply and discard for a generated candidate", async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();
    const onDiscard = vi.fn();
    render(
      withI18n(
        <GenerationTaskDropdown
          tasks={[
            {
              id: "run-ready",
              name: "Walk",
              prompt: "A relaxed walk",
              status: "awaiting_application",
              onApply,
              onDiscard,
            },
          ]}
        />,
      ),
    );

    await user.click(screen.getByRole("button", { name: /generation ready/ }));
    await user.click(screen.getByRole("button", { name: "Apply" }));
    expect(onApply).toHaveBeenCalledOnce();
    await user.click(screen.getByRole("button", { name: "Discard" }));
    expect(onDiscard).toHaveBeenCalledOnce();
  });
});
