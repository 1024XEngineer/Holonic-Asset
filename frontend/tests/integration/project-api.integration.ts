import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { projectApi } from "@/model/project/project.api";
import {
  startProjectApiServer,
  stopProjectApiServer,
  type ProjectApiServer,
} from "./support/project-api-server";

let server: ProjectApiServer;

beforeAll(async () => {
  server = await startProjectApiServer();
  vi.stubEnv("PUBLIC_CORE_API_BASE_URL", `${server.url}/api/v1`);
}, 60_000);

afterAll(async () => {
  vi.unstubAllEnvs();
  await stopProjectApiServer(server);
});

describe("project API integration", () => {
  it("supports the project CRUD lifecycle through the Core API", async () => {
    const created = await projectApi.create({
      name: "Integration Orchard",
      gameType: "Role-playing game",
      platform: "PC",
      description: "Restore the orchard.",
      reference: "reference.png",
      style: "pixel",
      perspective: "Top-Down",
      visualDirection: "reference.png",
    });
    expect(Number(created.id)).toBeGreaterThan(0);

    const listed = await projectApi.list();
    expect(listed.find(({ id }) => id === created.id)).toMatchObject({
      id: created.id,
      name: "Integration Orchard",
      description: "Restore the orchard.",
      visualDirection: "reference.png",
    });

    const detail = await projectApi.detail(created.id);
    expect(detail).toMatchObject({
      id: created.id,
      description: "Restore the orchard.",
      reference: "reference.png",
    });

    await expect(
      projectApi.update({
        ...detail,
        description: "Rebuild the moonlit orchard.",
        visualDirection: "updated-reference.png",
      }),
    ).resolves.toMatchObject({
      id: created.id,
      description: "Rebuild the moonlit orchard.",
    });
    await expect(projectApi.detail(created.id)).resolves.toMatchObject({
      id: created.id,
      name: "Integration Orchard",
      description: "Rebuild the moonlit orchard.",
      reference: "updated-reference.png",
      visualDirection: "updated-reference.png",
    });

    await projectApi.delete(created.id);
    const remaining = await projectApi.list();
    expect(remaining.some(({ id }) => id === created.id)).toBe(false);
  });
});
