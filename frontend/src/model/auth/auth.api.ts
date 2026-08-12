import { postEnvelope } from "@/model/fetchers";

import type { LoginRequest, LoginResponse } from "./auth.contract";

export const authApi = {
  login: (request: LoginRequest) =>
    postEnvelope<LoginResponse>("/auth/login", request),
};
