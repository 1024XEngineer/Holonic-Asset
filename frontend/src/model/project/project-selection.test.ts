import { afterEach, describe, expect, it, vi } from "vitest";

import type { ProjectSummary } from "./types";
import {
  reconcileProjectSelection,
  removeProjectSelection,
} from "./project-selection";

const projects = [project("first"), project("second")];

afterEach(() => vi.unstubAllGlobals());

describe("project selection", () => {
  it("remembers a requested project that exists", () => {
    const storage = createStorage();
    vi.stubGlobal("localStorage", storage);

    expect(reconcileProjectSelection(projects, "second")).toEqual({});
    expect(storage.setItem).toHaveBeenCalledWith(
      "game-asset-pack:last-project-id",
      "second",
    );
  });

  it("clears missing requests and stale remembered projects", () => {
    const storage = createStorage("missing");
    vi.stubGlobal("localStorage", storage);

    expect(reconcileProjectSelection(projects, "missing")).toEqual({});
    expect(storage.removeItem).toHaveBeenCalledOnce();

    storage.getItem.mockReturnValue("stale");
    expect(reconcileProjectSelection(projects, undefined)).toEqual({});
    expect(storage.removeItem).toHaveBeenCalledTimes(2);
  });

  it("redirects to a remembered project", () => {
    vi.stubGlobal("localStorage", createStorage("second"));

    expect(reconcileProjectSelection(projects, undefined)).toEqual({
      redirectProjectId: "second",
    });
  });

  it("works when storage is unavailable", () => {
    vi.stubGlobal("localStorage", {
      getItem: vi.fn(() => {
        throw new Error("blocked");
      }),
      setItem: vi.fn(() => {
        throw new Error("blocked");
      }),
      removeItem: vi.fn(() => {
        throw new Error("blocked");
      }),
    });

    expect(reconcileProjectSelection(projects, "first")).toEqual({});
    expect(reconcileProjectSelection(projects, undefined)).toEqual({});
    expect(removeProjectSelection(projects, "first", "first")).toBe("second");
  });

  it("selects a fallback only when the removed project was active", () => {
    const storage = createStorage("first");
    vi.stubGlobal("localStorage", storage);

    expect(removeProjectSelection(projects, "first", "first")).toBe("second");
    expect(removeProjectSelection(projects, "first", "second")).toBeUndefined();
  });
});

function project(id: string): ProjectSummary {
  return {
    id,
    name: id,
    style: "Top-Down",
    gameType: "Role-playing game",
    platform: "PC",
    description: "",
    reference: "",
    perspective: "Top-Down",
    assetCount: 0,
  };
}

function createStorage(initialValue: string | null = null) {
  let value = initialValue;
  return {
    getItem: vi.fn(() => value),
    setItem: vi.fn((_key: string, nextValue: string) => {
      value = nextValue;
    }),
    removeItem: vi.fn(() => {
      value = null;
    }),
  };
}
