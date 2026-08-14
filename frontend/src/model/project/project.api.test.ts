import { beforeEach, describe, expect, it, vi } from "vitest";

import { DataApiError } from "@/lib/data-api-error";
import type { CreateProjectInput, ProjectSummary } from "./types";
import type { ProjectResponse } from "./project.contract";

vi.mock("@/model/auth", () => ({
  readAuthenticatedUserId: () => 4927310,
}));

const mocks = vi.hoisted(() => ({
  core: {
    create: vi.fn(),
    delete: vi.fn(),
    detail: vi.fn(),
    generateReference: vi.fn(),
    list: vi.fn(),
    update: vi.fn(),
  },
}));

vi.mock("./core-project.api", () => ({ coreProjectApi: mocks.core }));
import { projectApi, toProjectSummary } from "./project.api";

const input: CreateProjectInput = {
  name: "Moonlit Orchard",
  gameType: "Role-playing game",
  platform: "PC",
  description: "Restore the orchard.",
  reference: "reference.png",
  style: "Top-Down",
  perspective: "Top-Down",
};

const remoteProject: ProjectResponse = {
  id: 7,
  name: input.name,
  gameType: "Role-playing game",
  targetPlatform: "PC",
  description: input.description,
  reference: input.reference,
  style: input.style,
  perspective: input.perspective,
  userID: 4927310,
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.core.create.mockResolvedValue({ id: 7 });
  mocks.core.detail.mockResolvedValue({ project: remoteProject });
  mocks.core.generateReference.mockResolvedValue({
    reference: "generated.png",
  });
  mocks.core.list.mockResolvedValue({ projects: [remoteProject] });
  mocks.core.update.mockResolvedValue({});
  mocks.core.delete.mockResolvedValue({});
});

describe("projectApi", () => {
  it("lists Core projects and returns an empty list when the API is unavailable", async () => {
    mocks.core.list.mockResolvedValue({
      projects: [remoteProject],
    });

    await expect(projectApi.list()).resolves.toEqual([
      toProjectSummary(remoteProject),
    ]);

    mocks.core.list.mockRejectedValueOnce(
      new DataApiError("UNAVAILABLE", "offline"),
    );
    await expect(projectApi.list()).resolves.toEqual([]);

    mocks.core.list.mockRejectedValueOnce(
      new DataApiError("UNKNOWN", "invalid response"),
    );
    await expect(projectApi.list()).rejects.toMatchObject({
      code: "UNKNOWN",
    });
  });

  it("uses Core detail, update, and deletion paths", async () => {
    await expect(projectApi.detail("7")).resolves.toEqual(
      toProjectSummary(remoteProject),
    );

    const remoteSummary = {
      ...toProjectSummary(remoteProject),
      reference: "updated-reference.png",
    };
    await expect(projectApi.update(remoteSummary)).resolves.toEqual(
      remoteSummary,
    );
    expect(mocks.core.update).toHaveBeenCalledWith({
      projectID: 7,
      ...coreFields(remoteSummary),
      reference: "updated-reference.png",
    });

    const clearedRemoteSummary = {
      ...remoteSummary,
      reference: "",
    };
    await projectApi.update(clearedRemoteSummary);
    expect(mocks.core.update).toHaveBeenLastCalledWith({
      projectID: 7,
      ...coreFields(clearedRemoteSummary),
      reference: "",
    });

    await projectApi.delete("7");
    expect(mocks.core.delete).toHaveBeenCalledWith({ projectID: 7 });
  });

  it("maps project inputs for creation and reference generation", async () => {
    await expect(projectApi.create(input)).resolves.toEqual({
      ...input,
      id: "7",
    });
    expect(mocks.core.create).toHaveBeenCalledWith({
      userID: 4927310,
      ...coreFields(input),
    });

    await expect(projectApi.generateReference(input)).resolves.toBe(
      "generated.png",
    );
    await expect(projectApi.regenerateReference(input)).resolves.toBe(
      "generated.png",
    );
    expect(mocks.core.generateReference).toHaveBeenNthCalledWith(
      1,
      coreFields(input),
    );
    expect(mocks.core.generateReference).toHaveBeenNthCalledWith(2, {
      ...coreFields(input),
    });

    const customInput = { ...input, gameType: "Deck-building roguelike" };
    await projectApi.generateReference(customInput);
    expect(mocks.core.generateReference).toHaveBeenLastCalledWith(
      coreFields(customInput),
    );
  });

  it("does not submit an expired Qiniu S3 signature", async () => {
    const signedReference =
      "https://xe-6-2.s3.cn-east-1.qiniucs.com/uploads/reference.png" +
      "?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
      "&X-Amz-Credential=access%2Fscope" +
      "&X-Amz-Date=20260814T063100Z" +
      "&X-Amz-Expires=1800" +
      "&X-Amz-SignedHeaders=host" +
      "&X-Amz-Signature=signature";
    const signedInput = { ...input, reference: signedReference };

    await projectApi.generateReference(signedInput);

    expect(mocks.core.generateReference).toHaveBeenCalledWith(
      expect.objectContaining({
        reference:
          "https://xe-6-2.s3.cn-east-1.qiniucs.com/uploads/reference.png",
      }),
    );
  });

  it("preserves game types from the current string contract", () => {
    expect(toProjectSummary(remoteProject)).toMatchObject({
      id: "7",
      gameType: "Role-playing game",
      reference: input.reference,
    });
    expect(toProjectSummary(remoteProject)).not.toHaveProperty("assetCount");
    expect(toProjectSummary(remoteProject, 3)).toHaveProperty("assetCount", 3);
    expect(
      toProjectSummary({ ...remoteProject, gameType: "Rhythm game" }),
    ).toMatchObject({ gameType: "Rhythm game" });
    expect(
      toProjectSummary({ ...remoteProject, gameType: "", targetPlatform: "" }),
    ).toMatchObject({ gameType: "", platform: "" });
  });
});

function coreFields(project: CreateProjectInput | ProjectSummary) {
  return {
    name: project.name,
    gameType: project.gameType,
    perspective: project.perspective,
    targetPlatform: "PC",
    description: project.description,
    reference: project.reference,
    style: project.style,
  };
}
