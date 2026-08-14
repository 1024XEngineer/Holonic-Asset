// @vitest-environment happy-dom

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { CreateAnimationTrigger } from "./create-animation-trigger";

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
          {(openDialog) => <button onClick={openDialog}>Open form</button>}
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
});
