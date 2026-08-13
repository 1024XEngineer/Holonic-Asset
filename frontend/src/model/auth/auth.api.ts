import { publicCoreApiClient, unwrapApiResponse } from "@/model/fetchers";

import type { LoginRequest, LoginResponse } from "./auth.contract";

export const authApi = {
  login: async (request: LoginRequest) =>
    unwrapApiResponse<LoginResponse>(
      await publicCoreApiClient.POST("/auth/login", { body: request }),
    ),
};
