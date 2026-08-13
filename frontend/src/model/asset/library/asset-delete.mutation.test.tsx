// @vitest-environment happy-dom

import type { PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { assetKeys } from "./keys";
import type { AssetGroup } from "./types";

const mocks = vi.hoisted(() => ({
  delete: vi.fn(),
  listGroups: vi.fn(),
}));

vi.mock("@/model/auth", () => ({ readAuthenticatedUserId: () => 7 }));
vi.mock("./asset.api", () => ({
  assetApi: { delete: mocks.delete, listGroups: mocks.listGroups },
}));

import { useDeleteAssetMutation } from "./asset-delete.mutation";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.listGroups.mockResolvedValue(assetGroups);
});

describe("useDeleteAssetMutation", () => {
  it("removes the asset while the delete request is still pending", async () => {
    const request = deferred<void>();
    mocks.delete.mockReturnValue(request.promise);
    const { queryClient, result } = renderDeleteMutation();

    act(() => result.current.mutate({ projectId: "42", assetId: "8" }));

    await waitFor(() => expect(cachedAssetIds(queryClient)).toEqual(["9"]));
    expect(result.current.isPending).toBe(true);

    await act(() => request.resolve());
  });

  it("restores the removed asset when the delete request fails", async () => {
    mocks.delete.mockRejectedValue(new Error("delete failed"));
    const { queryClient, result } = renderDeleteMutation();

    act(() => result.current.mutate({ projectId: "42", assetId: "8" }));

    await waitFor(() => expect(result.current.isError).toBe(true));
    await waitFor(() =>
      expect(cachedAssetIds(queryClient)).toEqual(["8", "9"]),
    );
  });

  it("settles before the background library refresh finishes", async () => {
    const refresh = deferred<AssetGroup[]>();
    mocks.delete.mockResolvedValue(undefined);
    mocks.listGroups.mockReturnValue(refresh.promise);
    const { queryClient, result } = renderDeleteMutation();

    act(() => result.current.mutate({ projectId: "42", assetId: "8" }));

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(cachedAssetIds(queryClient)).toEqual(["9"]);
    expect(mocks.listGroups).toHaveBeenCalledOnce();

    await act(() => refresh.resolve(assetGroups));
  });
});

function renderDeleteMutation() {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  queryClient.setQueryDefaults(assetKeys.library(7, "42"), {
    queryFn: mocks.listGroups,
  });
  queryClient.setQueryData(assetKeys.library(7, "42"), assetGroups);
  const wrapper = ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(useDeleteAssetMutation, { wrapper });
  return { queryClient, result };
}

function cachedAssetIds(queryClient: QueryClient) {
  return (
    queryClient.getQueryData<AssetGroup[]>(assetKeys.library(7, "42")) ?? []
  ).flatMap((group) => group.assets.map((asset) => asset.id));
}

const assetGroups: AssetGroup[] = [
  {
    kind: "object",
    assets: [
      { id: "8", name: "Barrel" },
      { id: "9", name: "Crate" },
    ] as AssetGroup["assets"],
  },
];

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, reject, resolve };
}
