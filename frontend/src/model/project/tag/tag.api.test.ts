import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/model/fetchers", () => ({
  coreApiClient: {
    GET: mocks.list,
    POST: mocks.create,
    PUT: mocks.update,
    DELETE: mocks.delete,
  },
  unwrapApiResponse: (result: unknown) => result,
}));

import { tagApi } from "./tag.api";

const tagResponse = {
  tagId: 7,
  projectId: 42,
  name: "  village ",
  description: "  Buildings  ",
  color: "#0969DA",
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.list.mockResolvedValue({ tags: [tagResponse] });
  mocks.create.mockResolvedValue({ tag: tagResponse });
  mocks.update.mockResolvedValue({ tag: tagResponse });
  mocks.delete.mockResolvedValue({ success: true });
});

describe("tagApi", () => {
  it("lists and maps project tags", async () => {
    await expect(tagApi.list("42")).resolves.toEqual([
      {
        tagId: 7,
        projectId: 42,
        name: "village",
        description: "Buildings",
        color: "#0969DA",
      },
    ]);
    expect(mocks.list).toHaveBeenCalledWith("/projects/{project_id}/tags", {
      params: { path: { project_id: 42 } },
      cache: "no-store",
    });
  });

  it("creates and updates tags through the scoped API", async () => {
    await tagApi.create("42", {
      name: "village",
      description: "Buildings",
      color: "#0969DA",
    });
    expect(mocks.create).toHaveBeenCalledWith("/projects/{project_id}/tags", {
      params: { path: { project_id: 42 } },
      body: {
        name: "village",
        description: "Buildings",
        color: "#0969DA",
      },
    });

    await tagApi.update("42", 7, { description: "Updated" });
    expect(mocks.update).toHaveBeenCalledWith(
      "/projects/{project_id}/tags/{tag_id}",
      {
        params: { path: { project_id: 42, tag_id: 7 } },
        body: { description: "Updated" },
      },
    );
  });

  it("deletes tags and rejects non-persisted project ids", async () => {
    await tagApi.delete("42", 7);
    expect(mocks.delete).toHaveBeenCalledWith(
      "/projects/{project_id}/tags/{tag_id}",
      { params: { path: { project_id: 42, tag_id: 7 } } },
    );
    await expect(tagApi.list("new-project")).rejects.toThrow(
      "persisted Core API project",
    );
  });
});
