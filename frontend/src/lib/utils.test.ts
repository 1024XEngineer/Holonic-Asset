import { describe, expect, it } from "vitest";
import { cn } from "./utils";

describe("cn utility", () => {
  it("merges class names correctly", () => {
    expect(cn("bg-red-500", "text-white")).toBe("bg-red-500 text-white");
  });

  it("ignores falsy classes", () => {
    expect(cn("p-4", undefined, "flex")).toBe("p-4 flex");
  });
});
