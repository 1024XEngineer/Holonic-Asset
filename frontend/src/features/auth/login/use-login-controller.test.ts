import { beforeEach, describe, expect, it, vi } from "vitest";

import { DataApiError } from "@/lib/data-api-error";

const mocks = vi.hoisted(() => ({
  form: { handleSubmit: vi.fn() },
  formOptions: undefined as
    | {
        defaultValues: { username: string; password: string };
        onSubmit: (value: {
          value: { username: string; password: string };
        }) => Promise<void>;
      }
    | undefined,
  login: vi.fn(),
  navigate: vi.fn(),
  resolveAuthRedirect: vi.fn(),
  saveAuthSession: vi.fn(),
  stateIndex: 0,
  stateValues: [] as unknown[],
  setters: [] as ReturnType<typeof vi.fn>[],
}));

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useState: (initial: unknown) => {
      const index = mocks.stateIndex++;
      const value =
        index < mocks.stateValues.length ? mocks.stateValues[index] : initial;
      const setter = vi.fn();
      mocks.setters.push(setter);
      return [value, setter];
    },
  };
});

vi.mock("@tanstack/react-form", () => ({
  useForm: (options: typeof mocks.formOptions) => {
    mocks.formOptions = options;
    return mocks.form;
  },
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/model/auth", () => ({
  authApi: { login: mocks.login },
  resolveAuthRedirect: mocks.resolveAuthRedirect,
  saveAuthSession: mocks.saveAuthSession,
}));

import { useLoginController } from "./use-login-controller";

const loginResponse = {
  accessToken: "access-token",
  tokenType: "Bearer" as const,
  expiresIn: 3600,
  user: {
    id: 7,
    username: "artist",
    email: "artist@example.com",
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.formOptions = undefined;
  mocks.stateIndex = 0;
  mocks.stateValues.length = 0;
  mocks.setters.length = 0;
  mocks.login.mockResolvedValue(loginResponse);
  mocks.navigate.mockResolvedValue(undefined);
  mocks.resolveAuthRedirect.mockReturnValue("/projects/7");
});

describe("useLoginController", () => {
  it("exposes the initial form and password visibility controls", () => {
    const controller = useLoginController();

    expect(mocks.formOptions?.defaultValues).toEqual({
      username: "",
      password: "",
    });
    expect(controller.form).toBe(mocks.form);
    expect(controller.showPassword).toBe(false);
    expect(controller.submitError).toBeUndefined();

    controller.togglePassword();
    controller.clearSubmitError();

    const toggle = mocks.setters[0].mock.calls[0][0] as (
      visible: boolean,
    ) => boolean;
    expect(toggle(false)).toBe(true);
    expect(toggle(true)).toBe(false);
    expect(mocks.setters[1]).toHaveBeenCalledWith(undefined);
  });

  it("logs in with a trimmed username, saves the session, and redirects", async () => {
    useLoginController("/projects/7");

    await mocks.formOptions!.onSubmit({
      value: { username: "  artist  ", password: " secret " },
    });

    expect(mocks.setters[1]).toHaveBeenCalledWith(undefined);
    expect(mocks.login).toHaveBeenCalledWith({
      username: "artist",
      password: " secret ",
    });
    expect(mocks.saveAuthSession).toHaveBeenCalledWith(loginResponse);
    expect(mocks.resolveAuthRedirect).toHaveBeenCalledWith("/projects/7");
    expect(mocks.navigate).toHaveBeenCalledWith({
      href: "/projects/7",
      replace: true,
    });
  });

  it.each([
    [
      "invalid credentials",
      new DataApiError("UNAUTHORIZED", "invalid credentials"),
      "invalidCredentials",
    ],
    [
      "an unavailable API",
      new DataApiError("UNAVAILABLE", "service unavailable"),
      "unavailable",
    ],
    ["an unexpected failure", new Error("network failed"), "unknownError"],
  ] as const)(
    "maps %s to the matching form error",
    async (_, error, expected) => {
      mocks.login.mockRejectedValueOnce(error);
      useLoginController();

      await mocks.formOptions!.onSubmit({
        value: { username: "artist", password: "secret" },
      });

      expect(mocks.setters[1]).toHaveBeenNthCalledWith(1, undefined);
      expect(mocks.setters[1]).toHaveBeenNthCalledWith(2, expected);
      expect(mocks.saveAuthSession).not.toHaveBeenCalled();
      expect(mocks.navigate).not.toHaveBeenCalled();
    },
  );
});
