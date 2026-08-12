import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import { clearDeletedProjectCache } from "./project-delete.mutation";
import { updateProjectCache } from "./project-update.mutation";
import { projectKeys } from "./keys";
import type { ProjectSummary } from "./types";

describe("project cache", () => {
  it("updates list and detail entries together", () => {
    const queryClient = new QueryClient();
    const current = project({ name: "Before" });
    const updated = project({ name: "After" });
    queryClient.setQueryData(projectKeys.list(), [current]);
    queryClient.setQueryData(projectKeys.detail(current.id), current);

    updateProjectCache(queryClient, updated);

    expect(queryClient.getQueryData(projectKeys.list())).toEqual([updated]);
    expect(queryClient.getQueryData(projectKeys.detail(updated.id))).toEqual(
      updated,
    );
  });

  it("removes list and detail entries together", () => {
    const queryClient = new QueryClient();
    const deleted = project();
    queryClient.setQueryData(projectKeys.list(), [deleted]);
    queryClient.setQueryData(projectKeys.detail(deleted.id), deleted);

    clearDeletedProjectCache(queryClient, deleted.id);

    expect(queryClient.getQueryData(projectKeys.list())).toEqual([]);
    expect(
      queryClient.getQueryData(projectKeys.detail(deleted.id)),
    ).toBeUndefined();
  });
});

function project(overrides: Partial<ProjectSummary> = {}): ProjectSummary {
  return {
    id: "project-1",
    name: "Project",
    style: "Top-Down",
    gameType: "Role-playing game",
    platform: "PC",
    description: "Description",
    reference: "reference.png",
    perspective: "Top-Down",
    assetCount: 0,
    ...overrides,
  };
}
