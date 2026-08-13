import type { operations } from "@/model/generated/core-api";

type LoginOperation = operations["login"];

export type LoginRequest =
  LoginOperation["requestBody"]["content"]["application/json"];
export type LoginResponse =
  LoginOperation["responses"][200]["content"]["application/json"]["data"];
export type AuthUser = LoginResponse["user"];
