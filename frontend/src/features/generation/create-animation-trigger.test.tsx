// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { CreateAnimationTrigger } from "./create-animation-trigger";

afterEach(cleanup);

describe("CreateAnimationTrigger", () => {
  it("uses the asset perspective directions and submits production defaults", async () => {
    const user = userEvent.setup();
    const onGenerate = vi.fn();
    render(
      withI18n(
        <CreateAnimationTrigger
          perspective="Side-On"
          isGenerating={false}
          onGenerate={onGenerate}
        >
          {(openDialog) => (
            <button type="button" onClick={openDialog}>
              Open form
            </button>
          )}
        </CreateAnimationTrigger>,
      ),
    );

    await user.click(screen.getByRole("button", { name: "Open form" }));

    expect(screen.getByRole("button", { name: "Left" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Right" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Front" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Cancel" })).toBeNull();

    const submitButton = screen.getByRole("button", {
      name: "Generate animation",
    });
    expect(submitButton.className).toContain("w-full");

    await user.type(screen.getByLabelText("Animation name"), "Walk");
    await user.type(
      screen.getByLabelText("Creative brief"),
      "A relaxed looping walk",
    );
    await user.click(submitButton);

    expect(onGenerate).toHaveBeenCalledWith({
      animationName: "Walk",
      direction: "left",
      creativeBrief: "A relaxed looping walk",
      frameCount: 8,
      fps: 12,
      duration: 5,
    });
  });

  it("submits the selected direction and production settings", async () => {
    const user = userEvent.setup();
    const onGenerate = vi.fn();
    render(
      withI18n(
        <CreateAnimationTrigger
          perspective="Top-Down"
          isGenerating={false}
          onGenerate={onGenerate}
        >
          {(openDialog) => (
            <button type="button" onClick={openDialog}>
              Open form
            </button>
          )}
        </CreateAnimationTrigger>,
      ),
    );

    await user.click(screen.getByRole("button", { name: "Open form" }));
    await user.type(screen.getByLabelText("Animation name"), "Run");
    await user.type(screen.getByLabelText("Creative brief"), "A fast run");
    await user.click(screen.getByRole("button", { name: "Back" }));

    const frames = screen.getByRole("spinbutton", { name: /^Frames/ });
    const frameRate = screen.getByRole("spinbutton", {
      name: /^Frame rate/,
    });
    const duration = screen.getByRole("spinbutton", {
      name: /^Source duration/,
    });
    fireEvent.change(frames, { target: { value: "" } });
    expect(
      (
        screen.getByRole("button", {
          name: "Generate animation",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    fireEvent.change(frames, { target: { value: "16" } });
    fireEvent.change(frameRate, { target: { value: "24" } });
    fireEvent.change(duration, { target: { value: "8" } });
    await user.click(
      screen.getByRole("button", { name: "Generate animation" }),
    );

    expect(onGenerate).toHaveBeenCalledWith({
      animationName: "Run",
      direction: "back",
      creativeBrief: "A fast run",
      frameCount: 16,
      fps: 24,
      duration: 8,
    });
  });

  it("does not submit while another animation is being queued", async () => {
    const user = userEvent.setup();
    const onGenerate = vi.fn();
    render(
      withI18n(
        <CreateAnimationTrigger
          perspective="Side-On"
          isGenerating
          onGenerate={onGenerate}
        >
          {(openDialog) => (
            <button type="button" onClick={openDialog}>
              Open form
            </button>
          )}
        </CreateAnimationTrigger>,
      ),
    );

    await user.click(screen.getByRole("button", { name: "Open form" }));
    await user.type(screen.getByLabelText("Animation name"), "Walk");
    await user.type(screen.getByLabelText("Creative brief"), "A calm walk");
    const submitButton = screen.getByRole("button", {
      name: "Adding to queue...",
    }) as HTMLButtonElement;
    expect(submitButton.disabled).toBe(true);

    fireEvent.submit(submitButton.closest("form")!);
    expect(onGenerate).not.toHaveBeenCalled();
  });
});
