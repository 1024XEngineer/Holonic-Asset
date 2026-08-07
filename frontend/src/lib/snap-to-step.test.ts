import { describe, expect, it } from "vitest";
import { snapToStep } from "./snap-to-step";

describe("snapToStep", () => {
  it("rounds to the nearest positive step", () => {
    expect(snapToStep(13, 8)).toBe(16);
    expect(snapToStep(-13, 8)).toBe(-16);
  });

  it("leaves values unchanged when the step is not positive", () => {
    expect(snapToStep(13, 0)).toBe(13);
    expect(snapToStep(13, -8)).toBe(13);
  });
});
