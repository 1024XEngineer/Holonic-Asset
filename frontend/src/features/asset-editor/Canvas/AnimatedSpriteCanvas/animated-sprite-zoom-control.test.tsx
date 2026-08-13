// @vitest-environment happy-dom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { AnimatedSpriteZoomControl } from "./animated-sprite-zoom-control";

describe("AnimatedSpriteZoomControl", () => {
  it("shows the current zoom and applies a clamped input value", () => {
    const onScaleChange = vi.fn();
    render(
      withI18n(
        <AnimatedSpriteZoomControl
          scale={0.64}
          minScale={0.3}
          maxScale={2}
          onScaleChange={onScaleChange}
        />,
      ),
    );
    const input = screen.getByRole("spinbutton", {
      name: "Canvas zoom percentage",
    });

    expect((input as HTMLInputElement).value).toBe("64");
    fireEvent.change(input, { target: { value: "250" } });
    fireEvent.blur(input);

    expect((input as HTMLInputElement).value).toBe("200");
    expect(onScaleChange).toHaveBeenCalledWith(2);
  });
});
