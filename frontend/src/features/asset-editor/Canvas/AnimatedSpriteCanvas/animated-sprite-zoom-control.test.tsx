// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { AnimatedSpriteZoomControl } from "./animated-sprite-zoom-control";

describe("AnimatedSpriteZoomControl", () => {
  afterEach(() => cleanup());

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

  it("restores invalid input", () => {
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
    }) as HTMLInputElement;

    fireEvent.change(input, { target: { value: "" } });
    fireEvent.blur(input);
    expect(input.value).toBe("64");
    expect(onScaleChange).not.toHaveBeenCalled();
  });

  it("clamps to the minimum and submits on Enter", () => {
    const onScaleChange = vi.fn();
    const { rerender } = render(
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
    }) as HTMLInputElement;

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "10" } });
    fireEvent.blur(input);
    expect(onScaleChange).toHaveBeenCalledWith(0.3);
    rerender(
      withI18n(
        <AnimatedSpriteZoomControl
          scale={0.3}
          minScale={0.3}
          maxScale={2}
          onScaleChange={onScaleChange}
        />,
      ),
    );
    expect(input.value).toBe("30");

    fireEvent.focus(input);
    const blur = vi.spyOn(input, "blur");
    fireEvent.keyDown(input, { key: "Enter" });
    expect(blur).toHaveBeenCalledOnce();
  });

  it("keeps an edited value until the canvas reports a new zoom", () => {
    const onScaleChange = vi.fn();
    const { rerender } = render(
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
    }) as HTMLInputElement;

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "90" } });
    rerender(
      withI18n(
        <AnimatedSpriteZoomControl
          scale={0.72}
          minScale={0.3}
          maxScale={2}
          onScaleChange={onScaleChange}
        />,
      ),
    );
    expect(input.value).toBe("90");

    fireEvent.blur(input);
    rerender(
      withI18n(
        <AnimatedSpriteZoomControl
          scale={0.72}
          minScale={0.3}
          maxScale={2}
          onScaleChange={onScaleChange}
        />,
      ),
    );
    expect(input.value).toBe("72");
  });
});
