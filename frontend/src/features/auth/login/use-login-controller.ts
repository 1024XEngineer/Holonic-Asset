import { useState } from "react";
import { useForm } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";

import { DataApiError } from "@/lib/data-api-error";
import {
  authApi,
  AuthSessionPersistenceError,
  saveAuthSession,
} from "@/model/auth";

import { resolveAuthRedirect } from "./resolve-auth-redirect";

export type LoginError =
  | "invalidCredentials"
  | "persistenceError"
  | "unavailable"
  | "unknownError";

export function useLoginController(redirectTo?: string) {
  const navigate = useNavigate();
  const [showPassword, setShowPassword] = useState(false);
  const [submitError, setSubmitError] = useState<LoginError>();
  const form = useForm({
    defaultValues: { username: "", password: "" },
    onSubmit: async ({ value }) => {
      setSubmitError(undefined);
      try {
        const response = await authApi.login({
          username: value.username.trim(),
          password: value.password,
        });
        saveAuthSession(response);
        await navigate({
          href: resolveAuthRedirect(redirectTo),
          replace: true,
        });
      } catch (error) {
        setSubmitError(loginError(error));
      }
    },
  });

  return {
    clearSubmitError: () => setSubmitError(undefined),
    form,
    showPassword,
    submitError,
    togglePassword: () => setShowPassword((visible) => !visible),
  };
}

export type LoginController = ReturnType<typeof useLoginController>;

function loginError(error: unknown): LoginError {
  if (error instanceof AuthSessionPersistenceError) return "persistenceError";
  if (error instanceof DataApiError) {
    if (error.code === "UNAUTHORIZED") return "invalidCredentials";
    if (error.code === "UNAVAILABLE") return "unavailable";
  }
  return "unknownError";
}
