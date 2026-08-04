import { beforeEach, describe, expect, it } from "vitest";

import type { CreateProjectInput } from "../types";
import {
  createMockProject,
  listMockProjects,
  resetMockProjects,
} from "./project.mock";

const projectInput: CreateProjectInput = {
  name: "Moonlit Orchard",
  gameType: "Role-playing game",
  platform: "PC",
  description: "A second project with the same display name.",
  reference: "",
  style: "Pixel art",
  visualStyle: "Pixel art",
  visualDirection: "",
};

describe("createMockProject", () => {
  beforeEach(() => resetMockProjects());

  it("assigns a unique server-side ID to every project", async () => {
    const first = await createMockProject(projectInput);
    const second = await createMockProject(projectInput);

    expect(first.id).not.toBe(second.id);
    expect(first).toMatchObject(projectInput);
    expect(second).toMatchObject(projectInput);
    expect(first.assetCount).toBe(0);
    expect(await listMockProjects()).toContainEqual(second);
  });
});
