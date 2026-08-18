// @vitest-environment happy-dom

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { GenerationReviewBar } from "./generation-review-bar";

afterEach(cleanup);

describe("GenerationReviewBar", () => {
  it("offers apply and deny below a generated preview", async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();
    const onDeny = vi.fn();
    render(
      withI18n(
        <GenerationReviewBar
          review={{
            name: "Walk",
            prompt: "A relaxed walk",
            pendingCount: 1,
            isLoading: false,
            isUnavailable: false,
            isResolving: false,
            onApply,
            onDeny,
          }}
        />,
      ),
    );

    expect(
      screen.getByRole("region", { name: "Review generated content" }),
    ).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Apply" }));
    await user.click(screen.getByRole("button", { name: "Deny" }));
    expect(onApply).toHaveBeenCalledOnce();
    expect(onDeny).toHaveBeenCalledOnce();
  });

  it("blocks apply until the candidate preview is ready", () => {
    render(
      withI18n(
        <GenerationReviewBar
          review={{
            name: "Walk",
            prompt: "A relaxed walk",
            pendingCount: 2,
            isLoading: true,
            isUnavailable: false,
            isResolving: false,
            onApply: vi.fn(),
            onDeny: vi.fn(),
          }}
        />,
      ),
    );

    expect(screen.getByText("Loading generated content...")).toBeTruthy();
    expect(
      (screen.getByRole("button", { name: "Apply" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    expect(screen.getByText("2 results")).toBeTruthy();
  });
});
